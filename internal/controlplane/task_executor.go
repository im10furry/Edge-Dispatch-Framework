package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/darkinno/edge-dispatch-framework/internal/models"
	"github.com/darkinno/edge-dispatch-framework/internal/store"
)

type TaskExecutor struct {
	pg        *store.PGStore
	scheduler *Scheduler
	pollIntv  time.Duration
}

func NewTaskExecutor(pg *store.PGStore, scheduler *Scheduler) *TaskExecutor {
	return &TaskExecutor{
		pg:        pg,
		scheduler: scheduler,
		pollIntv:  5 * time.Second,
	}
}

func (te *TaskExecutor) Start(ctx context.Context) {
	slog.Info("task executor started", "poll_interval", te.pollIntv)
	ticker := time.NewTicker(te.pollIntv)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("task executor stopping")
			return
		case <-ticker.C:
			te.processPending(ctx)
		}
	}
}

func (te *TaskExecutor) processPending(ctx context.Context) {
	tasks, _, err := te.pg.ListTasks(ctx, "", string(models.TaskStatusPending), "", 10, 0)
	if err != nil {
		slog.Warn("task executor: failed to list pending tasks", "err", err)
		return
	}
	if len(tasks) == 0 {
		return
	}

	for _, task := range tasks {
		te.executeTask(ctx, task)
	}
}

func (te *TaskExecutor) executeTask(ctx context.Context, task *models.Task) {
	slog.Info("task executor: executing task", "task_id", task.TaskID, "type", task.Type)

	switch task.Type {
	case models.TaskTypePurge:
		te.executePurge(ctx, task)
	case models.TaskTypePrewarm:
		te.executePrewarm(ctx, task)
	case models.TaskTypeBlock:
		te.executeBlock(ctx, task)
	default:
		slog.Warn("task executor: unknown task type", "type", task.Type, "task_id", task.TaskID)
	}
}

func (te *TaskExecutor) executePurge(ctx context.Context, task *models.Task) {
	var params models.TaskParams
	if err := json.Unmarshal(task.Params, &params); err != nil {
		slog.Warn("task executor: invalid purge params", "task_id", task.TaskID, "err", err)
		te.markComplete(ctx, task.TaskID, models.TaskStatusFailed, nil, 0, 0)
		return
	}

	keys := make([]string, 0)
	if params.ObjectKey != "" {
		keys = append(keys, params.ObjectKey)
	}

	if len(keys) == 0 {
		slog.Warn("task executor: purge task has no keys", "task_id", task.TaskID)
		te.markComplete(ctx, task.TaskID, models.TaskStatusFailed, nil, 0, 0)
		return
	}

	targetNodes := te.resolveTargetNodes(ctx, params)
	if len(targetNodes) == 0 {
		slog.Warn("task executor: no target nodes for purge", "task_id", task.TaskID)
		te.markComplete(ctx, task.TaskID, models.TaskStatusFailed, nil, 0, 0)
		return
	}

	te.pg.UpdateTaskStatus(ctx, task.TaskID, models.TaskStatusRunning, nil, 0, len(targetNodes))

	type response struct {
		Total   int `json:"total"`
		Results []struct {
			Key    string `json:"key"`
			Status string `json:"status"`
			Error  string `json:"error,omitempty"`
		} `json:"results"`
	}

	concurrency := params.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	doneNodes := 0

	for _, node := range targetNodes {
		wg.Add(1)
		go func(n *models.Node) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if len(n.Endpoints) == 0 {
				return
			}

			endpoint := n.Endpoints[0].URL()
			body, _ := json.Marshal(map[string][]string{"keys": keys})

			reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			httpReq, err := http.NewRequestWithContext(reqCtx, "POST",
				endpoint+"/internal/push/purge", bytes.NewReader(body))
			if err != nil {
				slog.Warn("task executor: purge request build failed", "node", n.NodeID, "err", err)
				return
			}
			httpReq.Header.Set("Content-Type", "application/json")

			client := &http.Client{Timeout: 60 * time.Second}
			resp, err := client.Do(httpReq)
			if err != nil {
				slog.Warn("task executor: purge request failed", "node", n.NodeID, "err", err)
				return
			}
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
			if err != nil {
				slog.Warn("task executor: purge read response failed", "node", n.NodeID, "err", err)
				return
			}

			var respData response
			if err := json.Unmarshal(bodyBytes, &respData); err != nil {
				slog.Warn("task executor: purge parse response failed", "node", n.NodeID, "err", err)
				return
			}

			mu.Lock()
			doneNodes++
			mu.Unlock()

			slog.Info("task executor: purge node complete",
				"task_id", task.TaskID,
				"node", n.NodeID,
				"keys", respData.Total,
			)
		}(node)
	}
	wg.Wait()

	progress := 100
	if len(targetNodes) > 0 {
		progress = doneNodes * 100 / len(targetNodes)
	}

	result, _ := json.Marshal(map[string]any{
		"keys_purged":   len(keys),
		"nodes_purged":  doneNodes,
		"total_nodes":   len(targetNodes),
	})
	te.markComplete(ctx, task.TaskID, models.TaskStatusCompleted, result, progress, doneNodes)
}

func (te *TaskExecutor) executePrewarm(ctx context.Context, task *models.Task) {
	var params models.TaskParams
	if err := json.Unmarshal(task.Params, &params); err != nil {
		slog.Warn("task executor: invalid prewarm params", "task_id", task.TaskID, "err", err)
		te.markComplete(ctx, task.TaskID, models.TaskStatusFailed, nil, 0, 0)
		return
	}

	keys := make([]string, 0)
	if params.ObjectKey != "" {
		keys = append(keys, params.ObjectKey)
	}
	if len(keys) == 0 {
		slog.Warn("task executor: prewarm task has no keys", "task_id", task.TaskID)
		te.markComplete(ctx, task.TaskID, models.TaskStatusFailed, nil, 0, 0)
		return
	}

	targetNodes := te.resolveTargetNodes(ctx, params)
	if len(targetNodes) == 0 {
		slog.Warn("task executor: no target nodes for prewarm", "task_id", task.TaskID)
		te.markComplete(ctx, task.TaskID, models.TaskStatusFailed, nil, 0, 0)
		return
	}

	te.pg.UpdateTaskStatus(ctx, task.TaskID, models.TaskStatusRunning, nil, 0, len(targetNodes))

	concurrency := params.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	doneNodes := 0

	for _, node := range targetNodes {
		wg.Add(1)
		go func(n *models.Node) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if len(n.Endpoints) == 0 {
				return
			}

			endpoint := n.Endpoints[0].URL()
			body, _ := json.Marshal(map[string][]string{"keys": keys})

			reqCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
			defer cancel()

			httpReq, err := http.NewRequestWithContext(reqCtx, "POST",
				endpoint+"/internal/push/prewarm", bytes.NewReader(body))
			if err != nil {
				slog.Warn("task executor: prewarm request build failed", "node", n.NodeID, "err", err)
				return
			}
			httpReq.Header.Set("Content-Type", "application/json")

			client := &http.Client{Timeout: 120 * time.Second}
			resp, err := client.Do(httpReq)
			if err != nil {
				slog.Warn("task executor: prewarm request failed", "node", n.NodeID, "err", err)
				return
			}
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
			if err != nil {
				slog.Warn("task executor: prewarm read response failed", "node", n.NodeID, "err", err)
				return
			}

			var respData struct {
				Total   int `json:"total"`
				Results []struct {
					Key    string `json:"key"`
					Status string `json:"status"`
					Size   int64  `json:"size,omitempty"`
					Error  string `json:"error,omitempty"`
				} `json:"results"`
			}
			if err := json.Unmarshal(bodyBytes, &respData); err != nil {
				slog.Warn("task executor: prewarm parse response failed", "node", n.NodeID, "err", err)
				return
			}

			mu.Lock()
			doneNodes++
			mu.Unlock()

			slog.Info("task executor: prewarm node complete",
				"task_id", task.TaskID,
				"node", n.NodeID,
				"keys", respData.Total,
			)
		}(node)
	}
	wg.Wait()

	progress := 100
	if len(targetNodes) > 0 {
		progress = doneNodes * 100 / len(targetNodes)
	}

	result, _ := json.Marshal(map[string]any{
		"keys_prewarmed": len(keys),
		"nodes_prewarmed": doneNodes,
		"total_nodes":     len(targetNodes),
	})
	te.markComplete(ctx, task.TaskID, models.TaskStatusCompleted, result, progress, doneNodes)
}

func (te *TaskExecutor) executeBlock(ctx context.Context, task *models.Task) {
	var params models.TaskParams
	if err := json.Unmarshal(task.Params, &params); err != nil {
		slog.Warn("task executor: invalid block params", "task_id", task.TaskID, "err", err)
		te.markComplete(ctx, task.TaskID, models.TaskStatusFailed, nil, 0, 0)
		return
	}

	keys := make([]string, 0)
	if params.ObjectKey != "" {
		keys = append(keys, params.ObjectKey)
	}
	if len(keys) == 0 {
		slog.Warn("task executor: block task has no keys", "task_id", task.TaskID)
		te.markComplete(ctx, task.TaskID, models.TaskStatusFailed, nil, 0, 0)
		return
	}

	targetNodes := te.resolveTargetNodes(ctx, params)
	if len(targetNodes) == 0 {
		slog.Warn("task executor: no target nodes for block", "task_id", task.TaskID)
		te.markComplete(ctx, task.TaskID, models.TaskStatusFailed, nil, 0, 0)
		return
	}

	te.pg.UpdateTaskStatus(ctx, task.TaskID, models.TaskStatusRunning, nil, 0, len(targetNodes))

	concurrency := params.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	doneNodes := 0

	for _, node := range targetNodes {
		wg.Add(1)
		go func(n *models.Node) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if len(n.Endpoints) == 0 {
				return
			}

			endpoint := n.Endpoints[0].URL()
			body, _ := json.Marshal(map[string][]string{"keys": keys})

			reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			httpReq, err := http.NewRequestWithContext(reqCtx, "POST",
				endpoint+"/internal/push/purge", bytes.NewReader(body))
			if err != nil {
				slog.Warn("task executor: block request build failed", "node", n.NodeID, "err", err)
				return
			}
			httpReq.Header.Set("Content-Type", "application/json")

			client := &http.Client{Timeout: 60 * time.Second}
			resp, err := client.Do(httpReq)
			if err != nil {
				slog.Warn("task executor: block request failed", "node", n.NodeID, "err", err)
				return
			}
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
			if err != nil {
				slog.Warn("task executor: block read response failed", "node", n.NodeID, "err", err)
				return
			}

			var respData struct {
				Total   int `json:"total"`
				Results []struct {
					Key    string `json:"key"`
					Status string `json:"status"`
					Error  string `json:"error,omitempty"`
				} `json:"results"`
			}
			if err := json.Unmarshal(bodyBytes, &respData); err != nil {
				slog.Warn("task executor: block parse response failed", "node", n.NodeID, "err", err)
				return
			}

			mu.Lock()
			doneNodes++
			mu.Unlock()

			slog.Info("task executor: block node complete",
				"task_id", task.TaskID,
				"node", n.NodeID,
				"keys", respData.Total,
			)
		}(node)
	}
	wg.Wait()

	progress := 100
	if len(targetNodes) > 0 {
		progress = doneNodes * 100 / len(targetNodes)
	}

	result, _ := json.Marshal(map[string]any{
		"keys_blocked":  len(keys),
		"nodes_blocked": doneNodes,
		"total_nodes":   len(targetNodes),
	})
	te.markComplete(ctx, task.TaskID, models.TaskStatusCompleted, result, progress, doneNodes)
}

func (te *TaskExecutor) resolveTargetNodes(ctx context.Context, params models.TaskParams) []*models.Node {
	if len(params.TargetNodes) > 0 {
		nodes := make([]*models.Node, 0, len(params.TargetNodes))
		for _, nodeID := range params.TargetNodes {
			node, err := te.pg.GetNode(ctx, nodeID)
			if err != nil {
				slog.Warn("task executor: node not found", "node_id", nodeID, "err", err)
				continue
			}
			if node.Status != models.NodeStatusActive && node.Status != models.NodeStatusDegraded {
				continue
			}
			nodes = append(nodes, node)
		}
		return nodes
	}

	nodes, err := te.scheduler.nodeCache.GetActiveNodes(ctx)
	if err != nil {
		slog.Warn("task executor: failed to get active nodes", "err", err)
		return nil
	}

	if params.TargetScope == "region" || params.TargetScope == "all" {
		return nodes
	}

	return nodes
}

func (te *TaskExecutor) markComplete(ctx context.Context, taskID string, status models.TaskStatus, result json.RawMessage, progress, doneNodes int) {
	tag, _ := json.Marshal(result)
	if err := te.pg.UpdateTaskStatus(ctx, taskID, status, tag, progress, doneNodes); err != nil {
		slog.Warn("task executor: failed to update task status", "task_id", taskID, "err", err)
	}
	slog.Info("task executor: task completed", "task_id", taskID, "status", status)
}



package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/darkinno/edge-dispatch-framework/internal/models"
)

func skipIfNoPG(t *testing.T) *PGStore {
	t.Helper()
	connStr := os.Getenv("TEST_PG_URL")
	if connStr == "" {
		t.Skip("TEST_PG_URL not set, skipping integration test")
	}
	pg, err := NewPGStore(context.Background(), connStr)
	if err != nil {
		t.Fatalf("connect pg: %v", err)
	}
	t.Cleanup(func() { pg.Close() })
	return pg
}

func skipIfNoRedis(t *testing.T) *RedisStore {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set, skipping integration test")
	}
	r, err := NewRedisStore(context.Background(), addr, "")
	if err != nil {
		t.Fatalf("connect redis: %v", err)
	}
	t.Cleanup(func() { r.Close() })
	return r
}

// ─── Node CRUD ───

func TestPGStoreCreateNode(t *testing.T) {
	t.Parallel()
	pg := skipIfNoPG(t)
	ctx := context.Background()

	req := models.RegisterRequest{
		NodeName: "test-node-1",
		TenantID: "default",
		Endpoints: []models.Endpoint{
			{Scheme: "http", Host: "10.0.0.1", Port: 8080},
		},
		Region: "cn-sh",
		ISP:    "ctcc",
		Capabilities: models.Capabilities{
			InboundReachable: true,
			CacheDiskGB:      100,
			MaxUplinkMbps:    1000,
		},
	}

	node, err := pg.CreateNode(ctx, req, "test-token")
	if err != nil {
		t.Fatalf("CreateNode: %v", err)
	}
	if node.NodeID == "" {
		t.Error("expected non-empty NodeID")
	}
	if node.Status != models.NodeStatusRegistered {
		t.Errorf("status = %s, want REGISTERED", node.Status)
	}
	if node.Weight != 100 {
		t.Errorf("weight = %d, want 100", node.Weight)
	}
}

func TestPGStoreGetNode(t *testing.T) {
	t.Parallel()
	pg := skipIfNoPG(t)
	ctx := context.Background()

	req := models.RegisterRequest{
		NodeName: "test-node-get",
		TenantID: "default",
		Endpoints: []models.Endpoint{
			{Scheme: "http", Host: "10.0.0.2", Port: 8080},
		},
		Region: "cn-bj",
		ISP:    "ctcc",
	}
	node, err := pg.CreateNode(ctx, req, "token")
	if err != nil {
		t.Fatalf("CreateNode: %v", err)
	}

	got, err := pg.GetNode(ctx, node.NodeID)
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if got.NodeID != node.NodeID {
		t.Errorf("NodeID = %s, want %s", got.NodeID, node.NodeID)
	}
	if got.Name != "test-node-get" {
		t.Errorf("Name = %s, want test-node-get", got.Name)
	}
}

func TestPGStoreGetNodeNotFound(t *testing.T) {
	t.Parallel()
	pg := skipIfNoPG(t)
	ctx := context.Background()

	_, err := pg.GetNode(ctx, "n_nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent node")
	}
}

func TestPGStoreListActiveNodes(t *testing.T) {
	t.Parallel()
	pg := skipIfNoPG(t)
	ctx := context.Background()

	req := models.RegisterRequest{
		NodeName: "test-active-node",
		TenantID: "default",
		Endpoints: []models.Endpoint{
			{Scheme: "http", Host: "10.0.0.3", Port: 8080},
		},
		Region: "cn-gz",
		ISP:    "cmcc",
	}
	node, err := pg.CreateNode(ctx, req, "token")
	if err != nil {
		t.Fatalf("CreateNode: %v", err)
	}
	if err := pg.UpdateNodeStatus(ctx, node.NodeID, models.NodeStatusActive); err != nil {
		t.Fatalf("UpdateNodeStatus: %v", err)
	}

	nodes, err := pg.ListActiveNodes(ctx)
	if err != nil {
		t.Fatalf("ListActiveNodes: %v", err)
	}

	found := false
	for _, n := range nodes {
		if n.NodeID == node.NodeID {
			found = true
			break
		}
	}
	if !found {
		t.Error("created active node not found in ListActiveNodes")
	}
}

func TestPGStoreUpdateNodeStatus(t *testing.T) {
	t.Parallel()
	pg := skipIfNoPG(t)
	ctx := context.Background()

	req := models.RegisterRequest{
		NodeName: "test-status-update",
		TenantID: "default",
		Endpoints: []models.Endpoint{
			{Scheme: "http", Host: "10.0.0.4", Port: 8080},
		},
		Region: "cn-hz",
		ISP:    "cmcc",
	}
	node, err := pg.CreateNode(ctx, req, "token")
	if err != nil {
		t.Fatalf("CreateNode: %v", err)
	}

	if err := pg.UpdateNodeStatus(ctx, node.NodeID, models.NodeStatusActive); err != nil {
		t.Fatalf("UpdateNodeStatus: %v", err)
	}

	got, err := pg.GetNode(ctx, node.NodeID)
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if got.Status != models.NodeStatusActive {
		t.Errorf("status = %s, want ACTIVE", got.Status)
	}
}

func TestPGStoreUpdateNodeScores(t *testing.T) {
	t.Parallel()
	pg := skipIfNoPG(t)
	ctx := context.Background()

	req := models.RegisterRequest{
		NodeName: "test-scores",
		TenantID: "default",
		Endpoints: []models.Endpoint{
			{Scheme: "http", Host: "10.0.0.5", Port: 8080},
		},
		Region: "cn-sz",
		ISP:    "ctcc",
	}
	node, err := pg.CreateNode(ctx, req, "token")
	if err != nil {
		t.Fatalf("CreateNode: %v", err)
	}

	scores := models.NodeScores{
		ReachableScore: 0.95,
		HealthScore:    0.90,
		RiskScore:      0.05,
	}
	if err := pg.UpdateNodeScores(ctx, node.NodeID, scores); err != nil {
		t.Fatalf("UpdateNodeScores: %v", err)
	}

	got, err := pg.GetNode(ctx, node.NodeID)
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if got.Scores.ReachableScore != 0.95 {
		t.Errorf("reachable_score = %f, want 0.95", got.Scores.ReachableScore)
	}
}

func TestPGStoreCountByStatus(t *testing.T) {
	t.Parallel()
	pg := skipIfNoPG(t)
	ctx := context.Background()

	req := models.RegisterRequest{
		NodeName: "test-count",
		TenantID: "default",
		Endpoints: []models.Endpoint{
			{Scheme: "http", Host: "10.0.0.6", Port: 8080},
		},
		Region: "cn-sh",
		ISP:    "ctcc",
	}
	node, err := pg.CreateNode(ctx, req, "token")
	if err != nil {
		t.Fatalf("CreateNode: %v", err)
	}

	count, err := pg.CountByStatus(ctx, string(models.NodeStatusRegistered))
	if err != nil {
		t.Fatalf("CountByStatus: %v", err)
	}
	if count < 1 {
		t.Errorf("CountByStatus REGISTERED = %d, want >= 1", count)
	}

	_ = node
}

func TestPGStoreListNodeIDsByTenant(t *testing.T) {
	t.Parallel()
	pg := skipIfNoPG(t)
	ctx := context.Background()

	req := models.RegisterRequest{
		NodeName: "test-tenant-node",
		TenantID: "default",
		Endpoints: []models.Endpoint{
			{Scheme: "http", Host: "10.0.0.7", Port: 8080},
		},
		Region: "cn-sh",
		ISP:    "ctcc",
	}
	node, err := pg.CreateNode(ctx, req, "token")
	if err != nil {
		t.Fatalf("CreateNode: %v", err)
	}

	ids, err := pg.ListNodeIDsByTenant(ctx, "default")
	if err != nil {
		t.Fatalf("ListNodeIDsByTenant: %v", err)
	}

	found := false
	for _, id := range ids {
		if id == node.NodeID {
			found = true
			break
		}
	}
	if !found {
		t.Error("created node ID not found in ListNodeIDsByTenant")
	}
}

func TestPGStoreNodeDisableEnable(t *testing.T) {
	t.Parallel()
	pg := skipIfNoPG(t)
	ctx := context.Background()

	req := models.RegisterRequest{
		NodeName: "test-disable",
		TenantID: "default",
		Endpoints: []models.Endpoint{
			{Scheme: "http", Host: "10.0.0.8", Port: 8080},
		},
		Region: "cn-sh",
		ISP:    "ctcc",
	}
	node, err := pg.CreateNode(ctx, req, "token")
	if err != nil {
		t.Fatalf("CreateNode: %v", err)
	}
	if err := pg.UpdateNodeStatus(ctx, node.NodeID, models.NodeStatusActive); err != nil {
		t.Fatalf("UpdateNodeStatus: %v", err)
	}

	until := time.Now().Add(1 * time.Hour)
	if err := pg.NodeDisable(ctx, node.NodeID, "maintenance", "scheduled upgrade", until); err != nil {
		t.Fatalf("NodeDisable: %v", err)
	}

	got, err := pg.GetNode(ctx, node.NodeID)
	if err != nil {
		t.Fatalf("GetNode after disable: %v", err)
	}
	if got.Status != models.NodeStatusDisabled {
		t.Errorf("status after disable = %s, want DISABLED", got.Status)
	}
	if got.DisableReason != "maintenance" {
		t.Errorf("disable_reason = %s, want maintenance", got.DisableReason)
	}

	if err := pg.NodeEnable(ctx, node.NodeID); err != nil {
		t.Fatalf("NodeEnable: %v", err)
	}

	got, err = pg.GetNode(ctx, node.NodeID)
	if err != nil {
		t.Fatalf("GetNode after enable: %v", err)
	}
	if got.Status != models.NodeStatusActive {
		t.Errorf("status after enable = %s, want ACTIVE", got.Status)
	}
}

func TestPGStoreRevokeNode(t *testing.T) {
	t.Parallel()
	pg := skipIfNoPG(t)
	ctx := context.Background()

	req := models.RegisterRequest{
		NodeName: "test-revoke",
		TenantID: "default",
		Endpoints: []models.Endpoint{
			{Scheme: "http", Host: "10.0.0.9", Port: 8080},
		},
		Region: "cn-sh",
		ISP:    "ctcc",
	}
	node, err := pg.CreateNode(ctx, req, "token")
	if err != nil {
		t.Fatalf("CreateNode: %v", err)
	}

	if err := pg.RevokeNode(ctx, node.NodeID); err != nil {
		t.Fatalf("RevokeNode: %v", err)
	}

	_, err = pg.GetNode(ctx, node.NodeID)
	if err == nil {
		t.Error("expected error getting revoked node")
	}
}

func TestPGStoreListNodes(t *testing.T) {
	t.Parallel()
	pg := skipIfNoPG(t)
	ctx := context.Background()

	req := models.RegisterRequest{
		NodeName: "test-list",
		TenantID: "default",
		Endpoints: []models.Endpoint{
			{Scheme: "http", Host: "10.0.0.10", Port: 8080},
		},
		Region: "cn-sh",
		ISP:    "ctcc",
	}
	_, err := pg.CreateNode(ctx, req, "token")
	if err != nil {
		t.Fatalf("CreateNode: %v", err)
	}

	nodes, total, err := pg.ListNodes(ctx, "default", "", "", "", "", 10, 0)
	if err != nil {
		t.Fatalf("ListNodes: %v", err)
	}
	if total < 1 {
		t.Errorf("total = %d, want >= 1", total)
	}
	if len(nodes) < 1 {
		t.Error("expected at least 1 node")
	}
}

// ─── Probe Results ───

func TestPGStoreSaveProbeResult(t *testing.T) {
	t.Parallel()
	pg := skipIfNoPG(t)
	ctx := context.Background()

	pr := models.ProbeResult{
		NodeID:   "test-node-probe",
		Endpoint: models.Endpoint{Scheme: "http", Host: "10.0.0.1", Port: 8080},
		Success:  true,
		RTTMs:    5.5,
	}
	if err := pg.SaveProbeResult(ctx, pr); err != nil {
		t.Fatalf("SaveProbeResult: %v", err)
	}
}

func TestPGStoreBatchSaveProbeResults(t *testing.T) {
	t.Parallel()
	pg := skipIfNoPG(t)
	ctx := context.Background()

	results := []models.ProbeResult{
		{NodeID: "batch-1", Endpoint: models.Endpoint{Scheme: "http", Host: "10.0.0.2", Port: 8080}, Success: true, RTTMs: 2.1},
		{NodeID: "batch-1", Endpoint: models.Endpoint{Scheme: "http", Host: "10.0.0.3", Port: 8080}, Success: false, Error: "timeout"},
	}
	if err := pg.BatchSaveProbeResults(ctx, results); err != nil {
		t.Fatalf("BatchSaveProbeResults: %v", err)
	}
}

// ─── Tenant CRUD ───

func TestPGStoreTenantCRUD(t *testing.T) {
	t.Parallel()
	pg := skipIfNoPG(t)
	ctx := context.Background()

	tnt := &models.Tenant{
		Name:        "Test Tenant",
		Description: "A test tenant",
	}
	if err := pg.CreateTenant(ctx, tnt); err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}
	if tnt.TenantID == "" {
		t.Error("expected non-empty TenantID after create")
	}

	got, err := pg.GetTenant(ctx, tnt.TenantID)
	if err != nil {
		t.Fatalf("GetTenant: %v", err)
	}
	if got.Name != "Test Tenant" {
		t.Errorf("Name = %s, want Test Tenant", got.Name)
	}

	tnt.Name = "Updated Tenant"
	if err := pg.UpdateTenant(ctx, tnt); err != nil {
		t.Fatalf("UpdateTenant: %v", err)
	}

	tenants, err := pg.ListTenants(ctx)
	if err != nil {
		t.Fatalf("ListTenants: %v", err)
	}
	found := false
	for _, t2 := range tenants {
		if t2.TenantID == tnt.TenantID {
			found = true
			break
		}
	}
	if !found {
		t.Error("tenant not found in ListTenants")
	}

	if err := pg.DeleteTenant(ctx, tnt.TenantID); err != nil {
		t.Fatalf("DeleteTenant: %v", err)
	}
}

// ─── Project CRUD ───

func TestPGStoreProjectCRUD(t *testing.T) {
	t.Parallel()
	pg := skipIfNoPG(t)
	ctx := context.Background()

	tnt := &models.Tenant{Name: "Project-Tenant", Description: "tenant for project test"}
	if err := pg.CreateTenant(ctx, tnt); err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}
	defer pg.DeleteTenant(ctx, tnt.TenantID)

	proj := &models.Project{
		TenantID:    tnt.TenantID,
		Name:        "Test Project",
		Description: "A test project",
	}
	if err := pg.CreateProject(ctx, proj); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if proj.ProjectID == "" {
		t.Error("expected non-empty ProjectID")
	}

	got, err := pg.GetProject(ctx, proj.ProjectID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if got.Name != "Test Project" {
		t.Errorf("Name = %s, want Test Project", got.Name)
	}

	proj.Name = "Updated Project"
	if err := pg.UpdateProject(ctx, proj); err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}

	projects, err := pg.ListProjects(ctx, tnt.TenantID)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	found := false
	for _, p := range projects {
		if p.ProjectID == proj.ProjectID {
			found = true
			break
		}
	}
	if !found {
		t.Error("project not found in ListProjects")
	}

	if err := pg.DeleteProject(ctx, proj.ProjectID); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
}

// ─── User CRUD ───

func TestPGStoreUserCRUD(t *testing.T) {
	t.Parallel()
	pg := skipIfNoPG(t)
	ctx := context.Background()

	user := &models.User{
		TenantID:    "default",
		Email:       "testuser@example.com",
		DisplayName: "Test User",
	}
	if err := pg.CreateUser(ctx, user, "hashed-password-123"); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if user.UserID == "" {
		t.Error("expected non-empty UserID")
	}

	got, err := pg.GetUser(ctx, user.UserID)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if got.Email != "testuser@example.com" {
		t.Errorf("Email = %s, want testuser@example.com", got.Email)
	}

	gotByEmail, pwHash, err := pg.GetUserByEmail(ctx, "testuser@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if gotByEmail.UserID != user.UserID {
		t.Errorf("UserID mismatch in GetUserByEmail")
	}
	if pwHash != "hashed-password-123" {
		t.Errorf("password hash mismatch")
	}

	user.DisplayName = "Updated User"
	if err := pg.UpdateUser(ctx, user); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	users, err := pg.ListUsers(ctx, "default")
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	found := false
	for _, u := range users {
		if u.UserID == user.UserID {
			found = true
			break
		}
	}
	if !found {
		t.Error("user not found in ListUsers")
	}

	// Refresh token
	if err := pg.UpdateUserRefreshToken(ctx, user.UserID, "hashed-refresh-token"); err != nil {
		t.Fatalf("UpdateUserRefreshToken: %v", err)
	}
	uid, err := pg.GetUserIDByRefreshToken(ctx, "hashed-refresh-token")
	if err != nil {
		t.Fatalf("GetUserIDByRefreshToken: %v", err)
	}
	if uid != user.UserID {
		t.Errorf("userID by refresh token = %s, want %s", uid, user.UserID)
	}

	if err := pg.DeleteUser(ctx, user.UserID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
}

// ─── Admin Policy CRUD ───

func TestPGStoreAdminPolicyCRUD(t *testing.T) {
	t.Parallel()
	pg := skipIfNoPG(t)
	ctx := context.Background()

	policy := &models.AdminPolicy{
		TenantID:    "default",
		ProjectID:   "default",
		Name:        "Test Policy",
		Type:        "dispatch",
		Description: "A test policy",
	}
	if err := pg.CreateAdminPolicy(ctx, policy); err != nil {
		t.Fatalf("CreateAdminPolicy: %v", err)
	}
	if policy.PolicyID == "" {
		t.Error("expected non-empty PolicyID")
	}

	got, err := pg.GetAdminPolicy(ctx, policy.PolicyID)
	if err != nil {
		t.Fatalf("GetAdminPolicy: %v", err)
	}
	if got.Name != "Test Policy" {
		t.Errorf("Name = %s, want Test Policy", got.Name)
	}

	policy.Name = "Updated Policy"
	if err := pg.UpdateAdminPolicy(ctx, policy); err != nil {
		t.Fatalf("UpdateAdminPolicy: %v", err)
	}

	policies, err := pg.ListAdminPolicies(ctx, "default", "default")
	if err != nil {
		t.Fatalf("ListAdminPolicies: %v", err)
	}
	found := false
	for _, p := range policies {
		if p.PolicyID == policy.PolicyID {
			found = true
			break
		}
	}
	if !found {
		t.Error("policy not found in ListAdminPolicies")
	}

	if err := pg.PublishAdminPolicy(ctx, policy.PolicyID); err != nil {
		t.Fatalf("PublishAdminPolicy: %v", err)
	}

	// Test rollback
	if err := pg.RollbackPolicy(ctx, policy.PolicyID, 1, []byte("{}")); err != nil {
		t.Fatalf("RollbackPolicy: %v", err)
	}

	if err := pg.DeleteAdminPolicy(ctx, policy.PolicyID); err != nil {
		t.Fatalf("DeleteAdminPolicy: %v", err)
	}
}

// ─── Ingress CRUD ───

func TestPGStoreIngressCRUD(t *testing.T) {
	t.Parallel()
	pg := skipIfNoPG(t)
	ctx := context.Background()

	ingress := &models.Ingress{
		TenantID:  "default",
		ProjectID: "default",
		Name:      "Test Ingress",
		Type:      "302",
		Domain:    "cdn.example.com",
	}
	if err := pg.CreateIngress(ctx, ingress); err != nil {
		t.Fatalf("CreateIngress: %v", err)
	}
	if ingress.IngressID == "" {
		t.Error("expected non-empty IngressID")
	}

	got, err := pg.GetIngress(ctx, ingress.IngressID)
	if err != nil {
		t.Fatalf("GetIngress: %v", err)
	}
	if got.Name != "Test Ingress" {
		t.Errorf("Name = %s, want Test Ingress", got.Name)
	}

	ingress.Name = "Updated Ingress"
	if err := pg.UpdateIngress(ctx, ingress); err != nil {
		t.Fatalf("UpdateIngress: %v", err)
	}

	ingresses, err := pg.ListIngresses(ctx, "default", "default")
	if err != nil {
		t.Fatalf("ListIngresses: %v", err)
	}
	found := false
	for _, ing := range ingresses {
		if ing.IngressID == ingress.IngressID {
			found = true
			break
		}
	}
	if !found {
		t.Error("ingress not found in ListIngresses")
	}

	if err := pg.DeleteIngress(ctx, ingress.IngressID); err != nil {
		t.Fatalf("DeleteIngress: %v", err)
	}
}

// ─── Task CRUD ───

func TestPGStoreTaskCRUD(t *testing.T) {
	t.Parallel()
	pg := skipIfNoPG(t)
	ctx := context.Background()

	task := &models.Task{
		TenantID:   "default",
		ProjectID:  "default",
		CreatorID:  "u_test",
		Type:       "purge",
		Status:     "pending",
		TotalNodes: 10,
	}
	if err := pg.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if task.TaskID == "" {
		t.Error("expected non-empty TaskID")
	}

	got, err := pg.GetTask(ctx, task.TaskID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.Status != "pending" {
		t.Errorf("Status = %s, want pending", got.Status)
	}

	if err := pg.UpdateTaskStatus(ctx, task.TaskID, "done", []byte("{}"), 5, 3); err != nil {
		t.Fatalf("UpdateTaskStatus: %v", err)
	}

	tasks, total, err := pg.ListTasks(ctx, "default", "", "", 10, 0)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if total < 1 {
		t.Errorf("total = %d, want >= 1", total)
	}
	if len(tasks) < 1 {
		t.Error("expected at least 1 task")
	}

	_ = task
}

// ─── Audit Event ───

func TestPGStoreAuditEvent(t *testing.T) {
	t.Parallel()
	pg := skipIfNoPG(t)
	ctx := context.Background()

	event := &models.AuditEvent{
		TenantID:     "default",
		ProjectID:    "default",
		ActorID:      "u_test",
		ActorEmail:   "test@example.com",
		Action:       "node.create",
		ResourceType: "node",
		ResourceID:   "n_test123",
		Result:       "success",
		SourceIP:     "127.0.0.1",
	}
	if err := pg.CreateAuditEvent(ctx, event); err != nil {
		t.Fatalf("CreateAuditEvent: %v", err)
	}

	events, _, err := pg.QueryAuditEvents(ctx, models.AuditQuery{
		TenantID: "default",
		Action:   "node.create",
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("QueryAuditEvents: %v", err)
	}
	if len(events) < 1 {
		t.Error("expected at least 1 audit event")
	}
}

// ─── Dashboard ───

func TestPGStoreDashboard(t *testing.T) {
	t.Parallel()
	pg := skipIfNoPG(t)
	ctx := context.Background()

	d, err := pg.GetDashboardData(ctx, "default")
	if err != nil {
		t.Fatalf("GetDashboardData: %v", err)
	}
	if d == nil {
		t.Error("expected non-nil dashboard data")
	}
}

// ─── Pool & Maintenance ───

func TestPGStorePoolStat(t *testing.T) {
	t.Parallel()
	pg := skipIfNoPG(t)
	_ = pg.PoolStat()
	_ = pg.Pool()
	_ = pg.ReadPool()
}

// ─── Redis Store ───

func TestRedisStoreHeartbeat(t *testing.T) {
	rd := skipIfNoRedis(t)
	ctx := context.Background()

	hb := models.HeartbeatRequest{
		NodeID: "test-hb-node",
		TS:     time.Now().Unix(),
		Runtime: models.NodeRuntime{
			CPU:        45.0,
			MemMB:      512,
			DiskFreeGB: 50,
			Conn:       20,
		},
		Traffic: models.NodeTraffic{
			EgressMbps:  50.5,
			IngressMbps: 10.0,
			Err5xxRate:  0.01,
		},
		Cache: models.NodeCache{
			HitRatio: 0.85,
		},
	}

	ttl := 30 * time.Second
	if err := rd.SaveHeartbeat(ctx, hb, ttl); err != nil {
		t.Fatalf("SaveHeartbeat: %v", err)
	}

	got, err := rd.GetHeartbeat(ctx, "test-hb-node")
	if err != nil {
		t.Fatalf("GetHeartbeat: %v", err)
	}
	if got.NodeID != "test-hb-node" {
		t.Errorf("NodeID = %s, want test-hb-node", got.NodeID)
	}
	if got.Runtime.CPU != 45.0 {
		t.Errorf("CPU = %f, want 45.0", got.Runtime.CPU)
	}
}

func TestRedisStorePipelineHeartbeats(t *testing.T) {
	rd := skipIfNoRedis(t)
	ctx := context.Background()

	hbs := []models.HeartbeatRequest{
		{NodeID: "pipe-1", TS: time.Now().Unix()},
		{NodeID: "pipe-2", TS: time.Now().Unix()},
	}
	ttl := 30 * time.Second
	if err := rd.PipelineSaveHeartbeats(ctx, hbs, ttl); err != nil {
		t.Fatalf("PipelineSaveHeartbeats: %v", err)
	}

	nodes, err := rd.GetAllHeartbeatNodes(ctx)
	if err != nil {
		t.Fatalf("GetAllHeartbeatNodes: %v", err)
	}
	found1, found2 := false, false
	for _, n := range nodes {
		if n == "pipe-1" {
			found1 = true
		}
		if n == "pipe-2" {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Error("pipelined heartbeats not found")
	}

	hbMap, err := rd.GetAllHeartbeats(ctx, []string{"pipe-1", "pipe-2"})
	if err != nil {
		t.Fatalf("GetAllHeartbeats: %v", err)
	}
	if len(hbMap) < 2 {
		t.Errorf("GetAllHeartbeats returned %d heartbeats, want >= 2", len(hbMap))
	}
}

func TestRedisStoreHasHeartbeat(t *testing.T) {
	rd := skipIfNoRedis(t)
	ctx := context.Background()

	hb := models.HeartbeatRequest{NodeID: "has-hb", TS: time.Now().Unix()}
	if err := rd.SaveHeartbeat(ctx, hb, 30*time.Second); err != nil {
		t.Fatalf("SaveHeartbeat: %v", err)
	}

	if !rd.HasHeartbeat(ctx, "has-hb") {
		t.Error("HasHeartbeat returned false for node with heartbeat")
	}
	if rd.HasHeartbeat(ctx, "nonexistent") {
		t.Error("HasHeartbeat returned true for nonexistent node")
	}
}

func TestRedisStoreEventPubSub(t *testing.T) {
	rd := skipIfNoRedis(t)
	ctx := context.Background()

	event := models.Event{
		Type:    "node.online",
		NodeID:  "event-node",
		Message: "node came online",
	}
	if err := rd.PublishEvent(ctx, "test-channel", event); err != nil {
		t.Fatalf("PublishEvent: %v", err)
	}
}

func TestRedisStoreRevoke(t *testing.T) {
	rd := skipIfNoRedis(t)
	ctx := context.Background()

	if err := rd.MarkNodeRevoked(ctx, "revoked-node"); err != nil {
		t.Fatalf("MarkNodeRevoked: %v", err)
	}

	if !rd.IsNodeRevoked(ctx, "revoked-node") {
		t.Error("IsNodeRevoked returned false for revoked node")
	}
	if rd.IsNodeRevoked(ctx, "not-revoked") {
		t.Error("IsNodeRevoked returned true for non-revoked node")
	}

	if err := rd.UnrevokeNode(ctx, "revoked-node"); err != nil {
		t.Fatalf("UnrevokeNode: %v", err)
	}
	if rd.IsNodeRevoked(ctx, "revoked-node") {
		t.Error("IsNodeRevoked returned true after unrevoke")
	}
}

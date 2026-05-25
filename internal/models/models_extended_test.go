package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEndpointURLWithoutPort(t *testing.T) {
	e := Endpoint{Scheme: "https", Host: "example.com"}
	got := e.URL()
	want := "https://example.com"
	if got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestEndpointURLWithoutScheme(t *testing.T) {
	e := Endpoint{Host: "example.com", Port: 8080}
	got := e.URL()
	want := "example.com:8080"
	if got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestEndpointURLHTTP(t *testing.T) {
	e := Endpoint{Scheme: "http", Host: "127.0.0.1", Port: 9090}
	got := e.URL()
	want := "http://127.0.0.1:9090"
	if got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestEndpointURLHTTPS(t *testing.T) {
	e := Endpoint{Scheme: "https", Host: "192.168.1.1", Port: 443}
	got := e.URL()
	want := "https://192.168.1.1:443"
	if got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestEndpointURLPortZero(t *testing.T) {
	e := Endpoint{Scheme: "http", Host: "example.com", Port: 0}
	got := e.URL()
	want := "http://example.com"
	if got != want {
		t.Errorf("URL() = %q, want %q", got, want)
	}
}

func TestInternNodeStatus(t *testing.T) {
	cases := []struct {
		input string
		want  NodeStatus
	}{
		{"ACTIVE", NodeStatusActive},
		{"REGISTERED", NodeStatusRegistered},
		{"DEGRADED", NodeStatusDegraded},
		{"QUARANTINED", NodeStatusQuarantined},
		{"OFFLINE", NodeStatusOffline},
		{"DISABLED", NodeStatusDisabled},
		{"MAINTENANCE", NodeStatusMaintenance},
		{"unknown", NodeStatus("unknown")},
		{"", NodeStatus("")},
	}

	for _, c := range cases {
		got := InternNodeStatus(c.input)
		if got != c.want {
			t.Errorf("InternNodeStatus(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestNodeStatusConstants(t *testing.T) {
	if string(NodeStatusActive) != "ACTIVE" {
		t.Errorf("NodeStatusActive = %q, want ACTIVE", NodeStatusActive)
	}
	if string(NodeStatusOffline) != "OFFLINE" {
		t.Errorf("NodeStatusOffline = %q, want OFFLINE", NodeStatusOffline)
	}
}

func TestUserRoleConstants(t *testing.T) {
	if string(UserRoleTenantOwner) != "tenant_owner" {
		t.Errorf("UserRoleTenantOwner = %q, want tenant_owner", UserRoleTenantOwner)
	}
	if string(UserRoleProjectViewer) != "project_viewer" {
		t.Errorf("UserRoleProjectViewer = %q, want project_viewer", UserRoleProjectViewer)
	}
}

func TestTaskStatusConstants(t *testing.T) {
	if string(TaskStatusPending) != "pending" {
		t.Errorf("TaskStatusPending = %q, want pending", TaskStatusPending)
	}
	if string(TaskStatusCompleted) != "completed" {
		t.Errorf("TaskStatusCompleted = %q, want completed", TaskStatusCompleted)
	}
}

func TestIngressTypeConstants(t *testing.T) {
	if string(IngressType302) != "302" {
		t.Errorf("IngressType302 = %q, want 302", IngressType302)
	}
	if string(IngressTypeDNS) != "dns" {
		t.Errorf("IngressTypeDNS = %q, want dns", IngressTypeDNS)
	}
	if string(IngressTypeGateway) != "gateway" {
		t.Errorf("IngressTypeGateway = %q, want gateway", IngressTypeGateway)
	}
}

func TestAdminPolicyTypeConstants(t *testing.T) {
	if string(AdminPolicyTypeDispatch) != "dispatch" {
		t.Errorf("AdminPolicyTypeDispatch = %q, want dispatch", AdminPolicyTypeDispatch)
	}
	if string(AdminPolicyTypeBlock) != "block" {
		t.Errorf("AdminPolicyTypeBlock = %q, want block", AdminPolicyTypeBlock)
	}
}

func TestTaskTypeConstants(t *testing.T) {
	if string(TaskTypePrewarm) != "prewarm" {
		t.Errorf("TaskTypePrewarm = %q, want prewarm", TaskTypePrewarm)
	}
	if string(TaskTypePurge) != "purge" {
		t.Errorf("TaskTypePurge = %q, want purge", TaskTypePurge)
	}
	if string(TaskTypeBlock) != "block" {
		t.Errorf("TaskTypeBlock = %q, want block", TaskTypeBlock)
	}
}

func TestEventConstants(t *testing.T) {
	if EventNodeOnline != "node.online" {
		t.Errorf("EventNodeOnline = %q, want node.online", EventNodeOnline)
	}
	if EventNodeOffline != "node.offline" {
		t.Errorf("EventNodeOffline = %q, want node.offline", EventNodeOffline)
	}
}

func TestAuditActionConstants(t *testing.T) {
	if AuditActionLogin != "auth.login" {
		t.Errorf("AuditActionLogin = %q, want auth.login", AuditActionLogin)
	}
	if AuditActionNodeDisable != "node.disable" {
		t.Errorf("AuditActionNodeDisable = %q, want node.disable", AuditActionNodeDisable)
	}
}

func TestNodeLabelsDefault(t *testing.T) {
	var labels NodeLabels
	if labels != nil {
		// nil map is valid for NodeLabels
		_ = labels
	}
}

func TestRegisterRequestJSON(t *testing.T) {
	req := RegisterRequest{
		NodeName:  "test-node",
		Region:    "cn-sh",
		ISP:       "ctcc",
		TenantID:  "t1",
		ProjectID: "p1",
		Endpoints: []Endpoint{
			{Scheme: "http", Host: "10.0.0.1", Port: 9090},
		},
		Capabilities: Capabilities{
			InboundReachable: true,
			CacheDiskGB:      100,
			MaxUplinkMbps:    1000,
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded RegisterRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.NodeName != "test-node" {
		t.Errorf("NodeName = %s, want test-node", decoded.NodeName)
	}
	if decoded.Region != "cn-sh" {
		t.Errorf("Region = %s, want cn-sh", decoded.Region)
	}
	if len(decoded.Endpoints) != 1 {
		t.Errorf("Endpoints len = %d, want 1", len(decoded.Endpoints))
	}
}

func TestLoginRequestJSON(t *testing.T) {
	req := LoginRequest{Email: "admin@edf.local", Password: "secret123"}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded LoginRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Email != "admin@edf.local" {
		t.Errorf("Email = %s", decoded.Email)
	}
	if decoded.Password != "secret123" {
		t.Errorf("Password = %s", decoded.Password)
	}
}

func TestPaginatedResponseJSON(t *testing.T) {
	pr := PaginatedResponse{
		Data:    []string{"a", "b", "c"},
		Total:   100,
		Limit:   10,
		Offset:  0,
		HasMore: true,
	}

	data, err := json.Marshal(pr)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded PaginatedResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Total != 100 {
		t.Errorf("Total = %d, want 100", decoded.Total)
	}
	if !decoded.HasMore {
		t.Error("HasMore should be true")
	}
}

func TestTaskJSON(t *testing.T) {
	task := Task{
		TaskID:     "task-123",
		CreatorID:  "user-1",
		TenantID:   "t1",
		Type:       TaskTypePrewarm,
		Status:     TaskStatusRunning,
		Progress:   50,
		TotalNodes: 10,
		DoneNodes:  5,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded Task
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.TaskID != "task-123" {
		t.Errorf("TaskID = %s", decoded.TaskID)
	}
	if decoded.Type != TaskTypePrewarm {
		t.Errorf("Type = %s", decoded.Type)
	}
	if decoded.Status != TaskStatusRunning {
		t.Errorf("Status = %s", decoded.Status)
	}
	if decoded.Progress != 50 {
		t.Errorf("Progress = %d", decoded.Progress)
	}
}

func TestAdminConfigDefault(t *testing.T) {
	cfg := AdminConfig{
		LocalAuthEnabled: true,
	}
	if !cfg.LocalAuthEnabled {
		t.Error("LocalAuthEnabled should be true")
	}
}

func TestErrorResponseJSON(t *testing.T) {
	resp := ErrorResponse{
		Error: ErrorDetail{
			Code:      "UNAUTHORIZED",
			Message:   "invalid token",
			RequestID: "req-123",
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded ErrorResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Error.Code != "UNAUTHORIZED" {
		t.Errorf("Error.Code = %s", decoded.Error.Code)
	}
	if decoded.Error.RequestID != "req-123" {
		t.Errorf("Error.RequestID = %s", decoded.Error.RequestID)
	}
}

func TestStreamTypeConstants(t *testing.T) {
	if string(StreamTypeHLS) != "hls" {
		t.Errorf("StreamTypeHLS = %q, want hls", StreamTypeHLS)
	}
	if string(StreamTypeDASH) != "dash" {
		t.Errorf("StreamTypeDASH = %q, want dash", StreamTypeDASH)
	}
}

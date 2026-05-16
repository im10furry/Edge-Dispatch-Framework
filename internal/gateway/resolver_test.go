package gateway

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestControlPlaneClientCachesDispatchToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/dispatch/resolve" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(DispatchResponse{
			Token: DispatchToken{Value: "edge-token"},
			Candidates: []CandidateInfo{
				{ID: "node-1", Endpoint: "http://127.0.0.1:9090", Weight: 100},
			},
		})
		if err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer ts.Close()

	client := NewControlPlaneClient(ts.URL, "cp-token", slog.Default())
	nodeID, err := client.GetBestNode("video/test.mp4", "203.0.113.10")
	if err != nil {
		t.Fatalf("GetBestNode: %v", err)
	}
	if nodeID != "node-1" {
		t.Fatalf("nodeID = %q, want node-1", nodeID)
	}

	token, ok := client.GetEdgeToken("node-1")
	if !ok {
		t.Fatal("expected cached token")
	}
	if token != "edge-token" {
		t.Fatalf("token = %q, want edge-token", token)
	}
}

func TestSetTokenQuery(t *testing.T) {
	got := setTokenQuery("a=1&token=old", "new value")
	if got != "a=1&token=new+value" {
		t.Fatalf("query = %q, want a=1&token=new+value", got)
	}
}

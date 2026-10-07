package raft

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsExposeRaftStateAndRPCMeasurements(t *testing.T) {
	node := NewRaft(1, []int{0, 1, 2})

	metrics, err := NewMetrics(node)
	if err != nil {
		t.Fatalf("NewMetrics returned an error: %v", err)
	}

	finish := metrics.observeRPC("AppendEntries")
	finish()

	response := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(response, httptest.NewRequest("GET", "/metrics", nil))

	if response.Code != 200 {
		t.Fatalf("metrics endpoint returned status %d", response.Code)
	}

	for _, metric := range []string{
		"raft_current_term",
		"raft_is_leader",
		"raft_commit_index",
		"raft_log_entries",
		"raft_rpc_requests_total",
		"raft_rpc_request_duration_seconds_count",
	} {
		if !strings.Contains(response.Body.String(), metric) {
			t.Errorf("metrics output does not contain %q", metric)
		}
	}

	if !strings.Contains(response.Body.String(), `node_id="1"`) {
		t.Error("metrics output does not identify node 1")
	}
}

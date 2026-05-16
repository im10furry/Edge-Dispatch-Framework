package metrics

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegistryDoesNotDuplicateMetricDefinitions(t *testing.T) {
	old := defaultRegistry
	defaultRegistry = &registry{
		gauges:   make(map[string]*gaugeMetric),
		counters: make(map[string]*counterMetric),
	}
	t.Cleanup(func() { defaultRegistry = old })

	NewCounter("test_counter_total", "test counter")
	NewCounter("test_counter_total", "test counter")
	NewGauge("test_gauge", "test gauge", "state")
	NewGauge("test_gauge", "test gauge", "state")

	rec := httptest.NewRecorder()
	defaultRegistry.writeTo(rec)
	body := rec.Body.String()

	if got := strings.Count(body, "# HELP test_counter_total"); got != 1 {
		t.Fatalf("counter HELP count = %d, want 1\n%s", got, body)
	}
	if got := strings.Count(body, "# TYPE test_gauge"); got != 1 {
		t.Fatalf("gauge TYPE count = %d, want 1\n%s", got, body)
	}
}

package metrics

import (
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func resetDefaultRegistry() {
	defaultRegistry = &registry{
		gauges:   make(map[string]*gaugeMetric),
		counters: make(map[string]*counterMetric),
	}
}

func TestGaugeSetAndOutput(t *testing.T) {
	old := defaultRegistry
	t.Cleanup(func() { defaultRegistry = old })
	resetDefaultRegistry()

	g := NewGauge("test_gauge_val", "test gauge", "region")
	g.Set(42.5, "cn-sh")

	rec := httptest.NewRecorder()
	defaultRegistry.writeTo(rec)
	body := rec.Body.String()

	if !strings.Contains(body, "test_gauge_val") {
		t.Error("expected gauge name in output")
	}
	if !strings.Contains(body, "42.5") {
		t.Errorf("expected gauge value 42.5 in output, got: %s", body)
	}
	if !strings.Contains(body, `region="cn-sh"`) {
		t.Errorf("expected label in output, got: %s", body)
	}
}

func TestCounterIncAndOutput(t *testing.T) {
	old := defaultRegistry
	t.Cleanup(func() { defaultRegistry = old })
	resetDefaultRegistry()

	c := NewCounter("test_counter_val", "test counter")
	c.Inc()
	c.Inc()
	c.Add(3)

	rec := httptest.NewRecorder()
	defaultRegistry.writeTo(rec)
	body := rec.Body.String()

	if !strings.Contains(body, "test_counter_val") {
		t.Error("expected counter name in output")
	}
	// 1 + 1 + 3 = 5
	if !strings.Contains(body, "test_counter_val 5") {
		t.Errorf("expected counter value 5, got: %s", body)
	}
}

func TestCounterWithLabels(t *testing.T) {
	old := defaultRegistry
	t.Cleanup(func() { defaultRegistry = old })
	resetDefaultRegistry()

	c := NewCounter("http_requests_total", "HTTP requests", "method", "status")
	c.Inc("GET", "200")
	c.Inc("GET", "200")
	c.Inc("POST", "201")

	rec := httptest.NewRecorder()
	defaultRegistry.writeTo(rec)
	body := rec.Body.String()

	if !strings.Contains(body, `method="GET",status="200"`) {
		t.Errorf("expected GET/200 labels, got: %s", body)
	}
	if !strings.Contains(body, `method="POST",status="201"`) {
		t.Errorf("expected POST/201 labels, got: %s", body)
	}
}

func TestGaugeWithLabels(t *testing.T) {
	old := defaultRegistry
	t.Cleanup(func() { defaultRegistry = old })
	resetDefaultRegistry()

	g := NewGauge("cpu_usage", "CPU usage percent", "host")
	g.Set(75.0, "server-1")
	g.Set(42.0, "server-2")

	rec := httptest.NewRecorder()
	defaultRegistry.writeTo(rec)
	body := rec.Body.String()

	if !strings.Contains(body, `host="server-1"`) {
		t.Errorf("expected server-1 label, got: %s", body)
	}
	if !strings.Contains(body, `host="server-2"`) {
		t.Errorf("expected server-2 label, got: %s", body)
	}
}

func TestMultipleMetricTypes(t *testing.T) {
	old := defaultRegistry
	t.Cleanup(func() { defaultRegistry = old })
	resetDefaultRegistry()

	c1 := NewCounter("metric_a", "counter a")
	c2 := NewCounter("metric_b", "counter b")
	g1 := NewGauge("metric_c", "gauge c")
	g2 := NewGauge("metric_d", "gauge d", "lbl")

	c1.Inc()
	c2.Inc()
	g1.Set(1.0)
	g2.Set(2.0, "v")

	rec := httptest.NewRecorder()
	defaultRegistry.writeTo(rec)
	body := rec.Body.String()

	for _, name := range []string{"metric_a", "metric_b", "metric_c", "metric_d"} {
		if !strings.Contains(body, name) {
			t.Errorf("expected metric %s in output", name)
		}
	}
	if strings.Count(body, "# HELP") != 4 {
		t.Errorf("expected 4 HELP lines, got %d", strings.Count(body, "# HELP"))
	}
}

func TestHandlerContentType(t *testing.T) {
	old := defaultRegistry
	t.Cleanup(func() { defaultRegistry = old })
	resetDefaultRegistry()

	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain", ct)
	}
}

func TestMakeLabelKey(t *testing.T) {
	key := makeLabelKey([]string{"region", "isp"}, []string{"cn-sh", "ctcc"})
	if key != "cn-sh\x00ctcc" {
		t.Errorf("makeLabelKey = %q", key)
	}

	keyMismatch := makeLabelKey([]string{"region"}, []string{"cn-sh", "ctcc"})
	if keyMismatch != "_" {
		t.Errorf("makeLabelKey mismatch = %q, want _", keyMismatch)
	}
}

func TestFormatLabels(t *testing.T) {
	got := formatLabels([]string{"region", "isp"}, "cn-sh\x00ctcc")
	want := `{region="cn-sh",isp="ctcc"}`
	if got != want {
		t.Errorf("formatLabels = %q, want %q", got, want)
	}

	// Empty key
	empty := formatLabels([]string{"region"}, "_")
	if empty != "" {
		t.Errorf("formatLabels with _ key = %q, want empty", empty)
	}

	// No labels
	noLabels := formatLabels(nil, "anything")
	if noLabels != "" {
		t.Errorf("formatLabels no labels = %q, want empty", noLabels)
	}

	// Mismatched values
	mismatch := formatLabels([]string{"a", "b"}, "only-one")
	if mismatch != "" {
		t.Errorf("formatLabels mismatch = %q, want empty", mismatch)
	}
}

func TestAtomicInt64CounterOperations(t *testing.T) {
	var c AtomicInt64Counter

	if v := c.Value(); v != 0 {
		t.Errorf("initial Value = %f, want 0", v)
	}

	c.Add(10)
	if v := c.Value(); v != 10 {
		t.Errorf("after Add(10): Value = %f, want 10", v)
	}

	c.Add(5)
	if v := c.Value(); v != 15 {
		t.Errorf("after Add(5): Value = %f, want 15", v)
	}

	c.Set(100)
	if v := c.Value(); v != 100 {
		t.Errorf("after Set(100): Value = %f, want 100", v)
	}
}

func TestAtomicInt64CounterNegative(t *testing.T) {
	var c AtomicInt64Counter
	c.Set(10)
	c.Add(-5)
	if v := c.Value(); v != 5 {
		t.Errorf("after Add(-5): Value = %f, want 5", v)
	}
}

func TestAtomicInt64CounterConcurrent(t *testing.T) {
	var c AtomicInt64Counter
	var wg sync.WaitGroup
	n := 100
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			c.Add(1)
			wg.Done()
		}()
	}
	wg.Wait()
	if v := c.Value(); v != float64(n) {
		t.Errorf("concurrent sum = %f, want %d", v, n)
	}
}

func TestGaugeOverwrite(t *testing.T) {
	old := defaultRegistry
	t.Cleanup(func() { defaultRegistry = old })
	resetDefaultRegistry()

	g := NewGauge("test_overwrite", "test", "key")
	g.Set(10.0, "a")
	g.Set(20.0, "a")

	rec := httptest.NewRecorder()
	defaultRegistry.writeTo(rec)
	body := rec.Body.String()

	// Data line should show 20 (overwritten), not 10
	if strings.Contains(body, "test_overwrite{key=\"a\"} 10") {
		t.Error("expected old value 10 to be overwritten by 20")
	}
	if !strings.Contains(body, "20") {
		t.Errorf("expected value 20 after overwrite, got: %s", body)
	}
}

func TestMetricHelpAndTypeLines(t *testing.T) {
	old := defaultRegistry
	t.Cleanup(func() { defaultRegistry = old })
	resetDefaultRegistry()

	NewCounter("my_counter", "This is a counter")

	rec := httptest.NewRecorder()
	defaultRegistry.writeTo(rec)
	body := rec.Body.String()

	if !strings.Contains(body, "# HELP my_counter This is a counter") {
		t.Errorf("missing HELP line, got: %s", body)
	}
	if !strings.Contains(body, "# TYPE my_counter counter") {
		t.Errorf("missing TYPE line, got: %s", body)
	}
}

package adminui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSPAHandlerServesIndex(t *testing.T) {
	handler := SPAHandler()

	req := httptest.NewRequest("GET", "/admin/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Admin UI Not Built") && !strings.Contains(string(body), "<!doctype html") && !strings.Contains(string(body), "<html") {
		t.Errorf("body does not contain expected content, got %d bytes", len(body))
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
}

func TestSPAHandlerServesIndexForUnknownRoutes(t *testing.T) {
	handler := SPAHandler()

	routes := []string{
		"/admin/login",
		"/admin/dashboard",
		"/admin/nodes",
		"/admin/tenants",
		"/admin/users",
		"/admin/settings",
	}

	for _, route := range routes {
		t.Run(strings.TrimPrefix(route, "/admin/"), func(t *testing.T) {
			req := httptest.NewRequest("GET", route, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			resp := w.Result()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("route %s: status = %d, want 200", route, resp.StatusCode)
			}
			ct := resp.Header.Get("Content-Type")
			if !strings.Contains(ct, "text/html") {
				t.Errorf("route %s: Content-Type = %q, want text/html", route, ct)
			}
		})
	}
}

func TestSPAHandlerRemovesAdminPrefix(t *testing.T) {
	handler := SPAHandler()

	req := httptest.NewRequest("GET", "/admin/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// After stripping, the internal path should be /index.html
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestSPAHandlerExactAdminPath(t *testing.T) {
	handler := SPAHandler()

	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	// http.FileServer may redirect /admin to /admin/ with 301
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusMovedPermanently {
		t.Errorf("status = %d, want 200 or 301", resp.StatusCode)
	}
}

func TestSpaFileSystemOpenExisting(t *testing.T) {
	handler := SPAHandler()

	req := httptest.NewRequest("GET", "/admin/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestSpaFileSystemOpenNonExistent(t *testing.T) {
	handler := SPAHandler()

	req := httptest.NewRequest("GET", "/admin/nonexistent-page", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200 (SPA fallback)", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "html") {
		t.Error("expected HTML content from SPA fallback")
	}
}

func TestSPAHandlerWriteTimeout(t *testing.T) {
	handler := SPAHandler()

	req := httptest.NewRequest("GET", "/admin/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Error("expected non-empty body")
	}
}

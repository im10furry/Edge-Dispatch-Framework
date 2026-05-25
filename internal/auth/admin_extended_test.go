package auth

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAdminAuthVerifySuccess(t *testing.T) {
	accessKey := "ak-test"
	secretKey := "sk-test"
	nonceCache := NewNonceCache(time.Minute)
	defer nonceCache.Close()

	a := NewAdminAuth(accessKey, secretKey, 5*time.Minute, 30*time.Second)

	req := signAdminRequest(t, "GET", "/", accessKey, secretKey, "nonce-1")
	_, err := a.Verify(req, nonceCache)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestAdminAuthVerifyMissingHeaders(t *testing.T) {
	nonceCache := NewNonceCache(time.Minute)
	defer nonceCache.Close()

	a := NewAdminAuth("ak", "sk", 5*time.Minute, 30*time.Second)

	cases := []struct {
		name    string
		headers map[string]string
	}{
		{"no headers", map[string]string{}},
		{"missing keyid", map[string]string{
			"X-Admin-Timestamp": strconv.FormatInt(time.Now().Unix(), 10),
			"X-Admin-Nonce":     "n1",
			"X-Admin-Signature": "sig",
		}},
		{"missing timestamp", map[string]string{
			"X-Admin-KeyId":     "ak",
			"X-Admin-Nonce":     "n1",
			"X-Admin-Signature": "sig",
		}},
		{"missing nonce", map[string]string{
			"X-Admin-KeyId":     "ak",
			"X-Admin-Timestamp": strconv.FormatInt(time.Now().Unix(), 10),
			"X-Admin-Signature": "sig",
		}},
		{"missing signature", map[string]string{
			"X-Admin-KeyId":     "ak",
			"X-Admin-Timestamp": strconv.FormatInt(time.Now().Unix(), 10),
			"X-Admin-Nonce":     "n1",
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			for k, v := range c.headers {
				req.Header.Set(k, v)
			}
			_, err := a.Verify(req, nonceCache)
			if err == nil {
				t.Error("expected error for missing headers")
			}
		})
	}
}

func TestAdminAuthVerifyInvalidTimestamp(t *testing.T) {
	nonceCache := NewNonceCache(time.Minute)
	defer nonceCache.Close()

	a := NewAdminAuth("ak", "sk", 5*time.Minute, 30*time.Second)

	futureTS := time.Now().Add(2 * time.Minute).Unix()
	req := signAdminRequest(t, "GET", "/", "ak", "sk", "nonce-future")
	req.Header.Set("X-Admin-Timestamp", strconv.FormatInt(futureTS, 10))
	_, err := a.Verify(req, nonceCache)
	if err == nil {
		t.Error("expected error for future timestamp")
	}
}

func TestAdminAuthVerifyReusedNonce(t *testing.T) {
	accessKey := "ak-test"
	secretKey := "sk-test"
	nonceCache := NewNonceCache(time.Minute)
	defer nonceCache.Close()

	a := NewAdminAuth(accessKey, secretKey, 5*time.Minute, 30*time.Second)

	req := signAdminRequest(t, "GET", "/", accessKey, secretKey, "nonce-reuse")
	_, err := a.Verify(req, nonceCache)
	if err != nil {
		t.Fatalf("first Verify: %v", err)
	}

	req2 := signAdminRequest(t, "GET", "/", accessKey, secretKey, "nonce-reuse")
	_, err = a.Verify(req2, nonceCache)
	if err == nil {
		t.Error("expected error for reused nonce")
	}
}

func TestAdminAuthVerifyInvalidAccessKey(t *testing.T) {
	nonceCache := NewNonceCache(time.Minute)
	defer nonceCache.Close()

	a := NewAdminAuth("ak-real", "sk-real", 5*time.Minute, 30*time.Second)

	req := signAdminRequest(t, "GET", "/", "ak-wrong", "sk-real", "nonce-badkey")
	_, err := a.Verify(req, nonceCache)
	if err == nil {
		t.Error("expected error for invalid access key")
	}
}

func TestAdminAuthVerifyInvalidSignature(t *testing.T) {
	nonceCache := NewNonceCache(time.Minute)
	defer nonceCache.Close()

	a := NewAdminAuth("ak", "sk", 5*time.Minute, 30*time.Second)

	req := signAdminRequest(t, "GET", "/", "ak", "sk", "nonce-badsig")
	req.Header.Set("X-Admin-Signature", "invalid-signature-base64")

	_, err := a.Verify(req, nonceCache)
	if err == nil {
		t.Error("expected error for invalid signature")
	}
}

func TestAdminAuthVerifyInvalidTimestampFormat(t *testing.T) {
	nonceCache := NewNonceCache(time.Minute)
	defer nonceCache.Close()

	a := NewAdminAuth("ak", "sk", 5*time.Minute, 30*time.Second)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Admin-KeyId", "ak")
	req.Header.Set("X-Admin-Timestamp", "not-a-number")
	req.Header.Set("X-Admin-Nonce", "n1")
	req.Header.Set("X-Admin-Signature", "sig")

	_, err := a.Verify(req, nonceCache)
	if err == nil {
		t.Error("expected error for invalid timestamp format")
	}
}

func TestAdminAuthAddKeyAndIsEnabled(t *testing.T) {
	a := NewAdminAuth("", "", 5*time.Minute, 30*time.Second)
	if a.IsEnabled() {
		t.Error("IsEnabled should be false when no keys configured")
	}

	a.AddKey("ak", "sk")
	if !a.IsEnabled() {
		t.Error("IsEnabled should be true after adding key")
	}
}

func TestAdminAuthMiddlewareEnabled(t *testing.T) {
	nonceCache := NewNonceCache(time.Minute)
	defer nonceCache.Close()

	a := NewAdminAuth("ak", "sk", 5*time.Minute, 30*time.Second)

	called := false
	handler := a.Middleware(nonceCache)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := signAdminRequest(t, "GET", "/", "ak", "sk", "nonce-mw")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler should have been called with valid auth")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestAdminAuthMiddlewareDisabled(t *testing.T) {
	nonceCache := NewNonceCache(time.Minute)
	defer nonceCache.Close()

	a := NewAdminAuth("", "", 5*time.Minute, 30*time.Second)

	called := false
	handler := a.Middleware(nonceCache)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler should be called when auth is disabled")
	}
}

func TestAdminAuthMiddlewareUnauthorized(t *testing.T) {
	nonceCache := NewNonceCache(time.Minute)
	defer nonceCache.Close()

	a := NewAdminAuth("ak", "sk", 5*time.Minute, 30*time.Second)

	handler := a.Middleware(nonceCache)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called without auth")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestJWTSessionVerifyExpired(t *testing.T) {
	j := NewJWTSession("secret", -1) // immediate expiry
	token, _, err := j.Sign("u1", "t1", "user@test.com")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	time.Sleep(10 * time.Millisecond)

	_, err = j.Verify(token)
	if err == nil {
		t.Error("expected expired error")
	}
}

func TestJWTSessionVerifyWrongSecret(t *testing.T) {
	j1 := NewJWTSession("secret-a", 3600)
	j2 := NewJWTSession("secret-b", 3600)
	token, _, err := j1.Sign("u1", "t1", "user@test.com")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	_, err = j2.Verify(token)
	if err == nil {
		t.Error("expected wrong secret error")
	}
}

func TestJWTSessionVerifyTampered(t *testing.T) {
	j := NewJWTSession("secret", 3600)
	token, _, err := j.Sign("u1", "t1", "user@test.com")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	parts := strings.Split(token, ".")
	tampered := parts[0] + "." + parts[1] + ".badsig"
	_, err = j.Verify(tampered)
	if err == nil {
		t.Error("expected tampered signature error")
	}
}

func TestJWTSessionVerifyMalformed(t *testing.T) {
	j := NewJWTSession("secret", 3600)

	cases := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"one part", "abc"},
		{"two parts", "a.b"},
		{"four parts", "a.b.c.d"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := j.Verify(c.token)
			if err == nil {
				t.Error("expected error for malformed token")
			}
		})
	}
}

func TestJWTSessionVerifyInvalidBase64(t *testing.T) {
	j := NewJWTSession("secret", 3600)
	// Invalid base64 in header, payload, and signature
	_, err := j.Verify("!!!.!!!.!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestJWTMiddlewareValid(t *testing.T) {
	j := NewJWTSession("secret", 3600)
	token, _, _ := j.Sign("u1", "t1", "user@test.com")

	called := false
	handler := j.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Header.Get("X-Actor-UserId") != "u1" {
			t.Errorf("X-Actor-UserId = %q, want u1", r.Header.Get("X-Actor-UserId"))
		}
		if r.Header.Get("X-Actor-TenantId") != "t1" {
			t.Errorf("X-Actor-TenantId = %q, want t1", r.Header.Get("X-Actor-TenantId"))
		}
		if r.Header.Get("X-Actor-Email") != "user@test.com" {
			t.Errorf("X-Actor-Email = %q, want user@test.com", r.Header.Get("X-Actor-Email"))
		}
		if r.Header.Get("X-Request-Id") == "" {
			t.Error("X-Request-Id should be set")
		}
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler should be called with valid JWT")
	}
}

func TestJWTMiddlewareMissingBearer(t *testing.T) {
	j := NewJWTSession("secret", 3600)

	handler := j.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called without bearer")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestJWTMiddlewareInvalidToken(t *testing.T) {
	j := NewJWTSession("secret", 3600)

	handler := j.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with invalid token")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestNonceCacheCheckAndStore(t *testing.T) {
	nc := NewNonceCache(time.Minute)
	defer nc.Close()

	if !nc.CheckAndStore("nonce-1") {
		t.Error("first CheckAndStore should succeed")
	}
	if nc.CheckAndStore("nonce-1") {
		t.Error("second CheckAndStore with same nonce should fail")
	}
	if !nc.CheckAndStore("nonce-2") {
		t.Error("different nonce should succeed")
	}
}

// Helper to create a signed admin request
func signAdminRequest(t *testing.T, method, path, accessKey, secretKey, nonce string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	req.Header.Set("X-Admin-KeyId", accessKey)
	req.Header.Set("X-Admin-Timestamp", ts)
	req.Header.Set("X-Admin-Nonce", nonce)
	req.Header.Set("X-Admin-Signature", computeHMAC(secretKey, accessKey, ts, nonce))
	return req
}

func TestComputeHMAC(t *testing.T) {
	sig1 := computeHMAC("sk", "ak", "1234", "nonce1")
	sig2 := computeHMAC("sk", "ak", "1234", "nonce1")
	if sig1 != sig2 {
		t.Error("computeHMAC should be deterministic")
	}

	sig3 := computeHMAC("sk-other", "ak", "1234", "nonce1")
	if sig1 == sig3 {
		t.Error("different secret should produce different signature")
	}
}

func TestAdminAuthClockSkew(t *testing.T) {
	nonceCache := NewNonceCache(time.Minute)
	defer nonceCache.Close()

	a := NewAdminAuth("ak", "sk", 5*time.Minute, 30*time.Second)

	now := time.Now()
	req := signAdminRequestWithTS(t, "ak", "sk", "nonce-skew-ok", now.Add(20*time.Second).Unix())
	_, err := a.Verify(req, nonceCache)
	if err != nil {
		t.Errorf("should accept within clock skew: %v", err)
	}
}

func signAdminRequestWithTS(t *testing.T, accessKey, secretKey, nonce string, ts int64) *http.Request {
	t.Helper()
	req := httptest.NewRequest("GET", "/", nil)
	tsStr := strconv.FormatInt(ts, 10)
	req.Header.Set("X-Admin-KeyId", accessKey)
	req.Header.Set("X-Admin-Timestamp", tsStr)
	req.Header.Set("X-Admin-Nonce", nonce)
	req.Header.Set("X-Admin-Signature", computeHMAC(secretKey, accessKey, tsStr, nonce))
	return req
}

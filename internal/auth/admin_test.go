package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestGenerateRefreshTokenUsesRandomBytes(t *testing.T) {
	t1, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken first: %v", err)
	}
	t2, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken second: %v", err)
	}

	if !strings.HasPrefix(t1, "rt_") {
		t.Fatalf("token prefix = %q, want rt_", t1[:3])
	}
	if t1 == t2 {
		t.Fatal("expected unique refresh tokens")
	}
	payload := strings.TrimPrefix(t1, "rt_")
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("decode token payload: %v", err)
	}
	if len(raw) != 32 {
		t.Fatalf("token random bytes = %d, want 32", len(raw))
	}
}

func TestNonceCacheCloseIsIdempotent(t *testing.T) {
	nc := NewNonceCache(time.Minute)
	nc.Close()
	nc.Close()
}

func TestJWTVerifyRejectsInvalidHeader(t *testing.T) {
	j := NewJWTSession("secret", 3600)
	token, _, err := j.Sign("u1", "t1", "user@example.com")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token parts = %d, want 3", len(parts))
	}
	header, err := json.Marshal(jwtHeader{Alg: "none", Typ: "JWT"})
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	parts[0] = base64.RawURLEncoding.EncodeToString(header)

	if _, err := j.Verify(strings.Join(parts, ".")); err == nil {
		t.Fatal("expected invalid header error")
	}
}

func TestJWTSignVerify(t *testing.T) {
	j := NewJWTSession("secret", 3600)
	token, exp, err := j.Sign("u1", "t1", "user@example.com")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if exp <= time.Now().Unix() {
		t.Fatalf("exp = %d, want future", exp)
	}

	claims, err := j.Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.UserID != "u1" || claims.TenantID != "t1" || claims.Email != "user@example.com" {
		t.Fatalf("claims = %+v", claims)
	}
}

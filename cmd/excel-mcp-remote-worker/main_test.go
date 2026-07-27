package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddlewareRejectsUnauthenticated(t *testing.T) {
	handler := authMiddleware("secret-token", "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for a request with no credentials, got %d", rec.Code)
	}
}

func TestAuthMiddlewareAcceptsValidBearerToken(t *testing.T) {
	handler := authMiddleware("secret-token", "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for a valid bearer token, got %d", rec.Code)
	}
}

func TestAuthMiddlewareRejectsWrongBearerToken(t *testing.T) {
	handler := authMiddleware("secret-token", "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for an incorrect bearer token, got %d", rec.Code)
	}
}

func TestAuthMiddlewareAcceptsValidHMACSignature(t *testing.T) {
	secret := "shared-secret"
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	handler := authMiddleware("", secret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The handler must still be able to read the body after the
		// middleware consumed it to compute the signature.
		got := make([]byte, len(body))
		n, _ := r.Body.Read(got)
		if n != len(body) {
			t.Errorf("downstream handler could not re-read the body: got %d bytes, want %d", n, len(body))
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", signature)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for a valid HMAC signature, got %d", rec.Code)
	}
}

func TestAuthMiddlewareRejectsTamperedBodyWithValidLookingSignature(t *testing.T) {
	secret := "shared-secret"
	originalBody := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(originalBody)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	handler := authMiddleware("", secret, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tamperedBody := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"sys.rm_rf"}}`)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(tamperedBody))
	req.Header.Set("X-Hub-Signature-256", signature)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for a body that doesn't match its signature, got %d", rec.Code)
	}
}

// excel-mcp-remote-worker exposes the same MCP tool set as excel-mcp-server
// (cmd/excel-mcp-server) over MCP's Streamable HTTP transport instead of
// stdio, so a cloud-side "intent action" can call tools that must run on
// this local PC (live Excel via OLE, local file access) without a custom
// dispatch protocol - any standard MCP HTTP client works.
//
// This binary is meant to sit behind a private tunnel (Cloudflare Tunnel,
// Tailscale Funnel, etc.), never exposed directly to the public internet.
// See README.md's "Remote (cloud to local PC) access" section.
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/server"
	excelserver "github.com/negokaz/excel-mcp-server/internal/server"
)

var version = "dev"

func main() {
	addr := os.Getenv("EXCEL_MCP_REMOTE_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8765"
	}
	token := os.Getenv("EXCEL_MCP_REMOTE_TOKEN")
	secret := os.Getenv("EXCEL_MCP_REMOTE_SECRET")
	if token == "" && secret == "" {
		fmt.Fprintln(os.Stderr,
			"refusing to start: set EXCEL_MCP_REMOTE_TOKEN (bearer token) and/or EXCEL_MCP_REMOTE_SECRET "+
				"(HMAC-SHA256 request signing secret, adnanh/webhook-style) before running this worker. "+
				"See README.md's \"Remote (cloud to local PC) access\" section.")
		os.Exit(1)
	}

	excelSrv := excelserver.New(version)
	httpServer := server.NewStreamableHTTPServer(excelSrv.MCPServer())

	mux := http.NewServeMux()
	mux.Handle("/mcp", authMiddleware(token, secret, httpServer))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	log.Printf("excel-mcp-remote-worker listening on %s/mcp", addr)
	log.Printf("bind address is intended for loopback + a private tunnel only - do not expose this port directly to the internet")
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

// authMiddleware rejects any request that presents neither a valid bearer
// token nor a valid HMAC-SHA256 body signature. It fails closed: if a check
// is configured (its secret is non-empty) and the request doesn't satisfy
// it, and no other configured check passes either, the request is rejected.
func authMiddleware(token string, secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token != "" && isValidBearerToken(r, token) {
			next.ServeHTTP(w, r)
			return
		}
		if secret != "" {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read request body", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(strings.NewReader(string(body)))
			if isValidHMACSignature(secret, body, r.Header.Get("X-Hub-Signature-256")) {
				next.ServeHTTP(w, r)
				return
			}
		}
		log.Printf("rejected unauthorized request from %s", r.RemoteAddr)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

func isValidBearerToken(r *http.Request, token string) bool {
	const prefix = "Bearer "
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, prefix) {
		return false
	}
	presented := strings.TrimPrefix(auth, prefix)
	return subtle.ConstantTimeCompare([]byte(presented), []byte(token)) == 1
}

// isValidHMACSignature verifies a "sha256=<hex>" signature header the same
// way adnanh/webhook's hmac-sha256 trigger rule and GitHub's webhook
// signing do: HMAC-SHA256 of the raw request body, keyed by the shared
// secret, compared in constant time.
func isValidHMACSignature(secret string, body []byte, header string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	expectedSig, err := hex.DecodeString(strings.TrimPrefix(header, prefix))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(mac.Sum(nil), expectedSig)
}

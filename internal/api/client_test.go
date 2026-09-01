package api

import (
	"net/http"
	"testing"
)

func TestClientConfigUsesSecretPath(t *testing.T) {
	server := newAuthTestServer(t)
	path := server.store.GetSettings().ClientConfigPath
	missing := performJSONRequest(t, server, http.MethodGet, "/client/wrong", nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("wrong path returned %d", missing.Code)
	}
	response := performJSONRequest(t, server, http.MethodGet, "/client/"+path, nil)
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("client config failed: %d %s", response.Code, response.Body.String())
	}
}

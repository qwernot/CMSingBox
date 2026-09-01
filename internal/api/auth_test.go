package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"cmsingbox.local/cmsingbox/internal/storage"
)

func newAuthTestServer(t *testing.T) *Server {
	t.Helper()
	store, err := storage.NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	return NewServer(store, nil, nil, nil, "", 9090, "test", nil)
}

func performJSONRequest(t *testing.T, server *Server, method, path string, body any, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &payload)
	req.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	server.router.ServeHTTP(response, req)
	return response
}

func TestAuthenticationFlow(t *testing.T) {
	server := newAuthTestServer(t)

	unauthorized := performJSONRequest(t, server, http.MethodGet, "/api/settings", nil)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", unauthorized.Code)
	}

	badLogin := performJSONRequest(t, server, http.MethodPost, "/api/auth/login", map[string]string{"username": "admin", "password": "wrong"})
	if badLogin.Code != http.StatusUnauthorized {
		t.Fatalf("expected invalid credentials to return 401, got %d", badLogin.Code)
	}

	login := performJSONRequest(t, server, http.MethodPost, "/api/auth/login", map[string]string{"username": "admin", "password": "admin"})
	if login.Code != http.StatusOK {
		t.Fatalf("expected login success, got %d: %s", login.Code, login.Body.String())
	}
	cookies := login.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != sessionCookieName || !cookies[0].HttpOnly {
		t.Fatalf("expected secure session cookie, got %#v", cookies)
	}

	authorized := performJSONRequest(t, server, http.MethodGet, "/api/settings", nil, cookies[0])
	if authorized.Code != http.StatusOK {
		t.Fatalf("expected authenticated settings request, got %d", authorized.Code)
	}

	logout := performJSONRequest(t, server, http.MethodPost, "/api/auth/logout", nil, cookies[0])
	if logout.Code != http.StatusOK {
		t.Fatalf("expected logout success, got %d", logout.Code)
	}

	afterLogout := performJSONRequest(t, server, http.MethodGet, "/api/settings", nil, cookies[0])
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("expected logged out session to return 401, got %d", afterLogout.Code)
	}
}

package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func authenticatedCookie(t *testing.T, server *Server) *http.Cookie {
	t.Helper()
	login := performJSONRequest(t, server, http.MethodPost, "/api/auth/login", map[string]string{"username": "admin", "password": "admin"})
	if login.Code != http.StatusOK || len(login.Result().Cookies()) == 0 {
		t.Fatalf("login failed: %d %s", login.Code, login.Body.String())
	}
	return login.Result().Cookies()[0]
}

func TestBackupExportExcludesCredentials(t *testing.T) {
	server := newAuthTestServer(t)
	cookie := authenticatedCookie(t, server)
	request := httptest.NewRequest(http.MethodGet, "/api/backup", nil)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	server.router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("export failed: %d %s", response.Code, response.Body.String())
	}
	archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
	if err != nil {
		t.Fatalf("invalid zip: %v", err)
	}
	if len(archive.File) != 1 || archive.File[0].Name != "data.json" {
		t.Fatalf("unexpected archive entries: %#v", archive.File)
	}
	file, err := archive.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(file)
	file.Close()
	if strings.Contains(string(data), "password_hash") {
		t.Fatal("backup must not contain password hashes")
	}
}

func TestBackupImportPreservesCredentials(t *testing.T) {
	server := newAuthTestServer(t)
	cookie := authenticatedCookie(t, server)
	settings := server.store.GetSettings()
	settings.MixedPort = 4321
	payload := map[string]any{
		"subscriptions": []any{}, "manual_nodes": []any{}, "filters": []any{}, "rules": []any{},
		"rule_groups": []any{}, "settings": settings,
		"auth": map[string]string{"username": "attacker", "password_hash": "invalid"},
	}
	jsonData, _ := json.Marshal(payload)
	var archiveBuffer bytes.Buffer
	archive := zip.NewWriter(&archiveBuffer)
	entry, _ := archive.Create("data.json")
	_, _ = entry.Write(jsonData)
	_ = archive.Close()

	var requestBody bytes.Buffer
	form := multipart.NewWriter(&requestBody)
	file, _ := form.CreateFormFile("file", "backup.zip")
	_, _ = file.Write(archiveBuffer.Bytes())
	_ = form.Close()
	request := httptest.NewRequest(http.MethodPost, "/api/backup/import", &requestBody)
	request.Header.Set("Content-Type", form.FormDataContentType())
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	server.router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("import failed: %d %s", response.Code, response.Body.String())
	}
	if server.store.GetSettings().MixedPort != 4321 {
		t.Fatal("settings were not restored")
	}
	if server.store.GetAuthConfig().Username != "admin" {
		t.Fatal("import must preserve authentication configuration")
	}
}

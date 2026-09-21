package api

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoginBackgroundLifecycle(t *testing.T) {
	server := newAuthTestServer(t)
	login := performJSONRequest(t, server, http.MethodPost, "/api/auth/login", map[string]string{"username": "admin", "password": "admin"})
	cookie := login.Result().Cookies()[0]

	var imageData bytes.Buffer
	picture := image.NewRGBA(image.Rect(0, 0, 1, 1))
	picture.Set(0, 0, color.Black)
	if err := png.Encode(&imageData, picture); err != nil {
		t.Fatal(err)
	}

	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, _ := form.CreateFormFile("file", "background.png")
	_, _ = file.Write(imageData.Bytes())
	_ = form.Close()
	request := httptest.NewRequest(http.MethodPost, "/api/settings/login-background", &body)
	request.Header.Set("Content-Type", form.FormDataContentType())
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	server.router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("upload returned %d: %s", response.Code, response.Body.String())
	}

	background := performJSONRequest(t, server, http.MethodGet, "/api/login-background", nil)
	if background.Code != http.StatusOK || background.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("background returned %d %q", background.Code, background.Header().Get("Content-Type"))
	}

	deleted := performJSONRequest(t, server, http.MethodDelete, "/api/settings/login-background", nil, cookie)
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete returned %d", deleted.Code)
	}
	missing := performJSONRequest(t, server, http.MethodGet, "/api/login-background", nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", missing.Code)
	}
}

func TestLoginBackgroundRejectsInvalidFile(t *testing.T) {
	server := newAuthTestServer(t)
	login := performJSONRequest(t, server, http.MethodPost, "/api/auth/login", map[string]string{"username": "admin", "password": "admin"})

	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, _ := form.CreateFormFile("file", "background.jpg")
	_, _ = file.Write([]byte("not an image"))
	_ = form.Close()
	request := httptest.NewRequest(http.MethodPost, "/api/settings/login-background", &body)
	request.Header.Set("Content-Type", form.FormDataContentType())
	request.AddCookie(login.Result().Cookies()[0])
	response := httptest.NewRecorder()
	server.router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}
}

package main

import (
	"crypto/sha256"
	"html/template"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"cmsingbox.local/cmsingbox/internal/licensing"
)

func TestLoginAndIssue(t *testing.T) {
	_, privateKey, err := licensing.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	passwordSum := sha256.Sum256([]byte("strong-test-password"))
	s := &server{
		privateKey: privateKey, passwordSum: passwordSum, auditPath: t.TempDir() + "/audit.jsonl",
		sessions: make(map[string]session), tmpl: template.Must(template.New("page").Parse(pageHTML)),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.home)
	mux.HandleFunc("POST /login", s.login)
	mux.HandleFunc("POST /issue", s.issue)
	ts := httptest.NewServer(securityHeaders(mux))
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	response, err := client.PostForm(ts.URL+"/login", url.Values{"password": {"strong-test-password"}})
	if err != nil || response.StatusCode != http.StatusOK {
		t.Fatalf("login failed: %v status=%v", err, response.StatusCode)
	}
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	csrfMatch := regexp.MustCompile(`name="csrf" value="([a-f0-9]+)"`).FindStringSubmatch(string(body))
	if len(csrfMatch) != 2 {
		t.Fatal("csrf token not found")
	}
	response, err = client.PostForm(ts.URL+"/issue", url.Values{
		"csrf": {csrfMatch[1]}, "device_code": {"123456"}, "subscriptions": {"25"}, "license_id": {"customer-test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), "CMS1.") || !strings.Contains(string(body), "签发成功") {
		t.Fatalf("license not issued: status=%d body=%s", response.StatusCode, body)
	}
	audit, err := os.ReadFile(s.auditPath)
	if err != nil || !strings.Contains(string(audit), `"max_subscriptions":25`) {
		t.Fatalf("audit not written: %v %s", err, audit)
	}
}

func TestLoginAllowsBrowserOriginMismatch(t *testing.T) {
	passwordSum := sha256.Sum256([]byte("strong-test-password"))
	s := &server{
		passwordSum: passwordSum,
		sessions:    make(map[string]session),
		tmpl:        template.Must(template.New("page").Parse(pageHTML)),
	}
	form := url.Values{"password": {"strong-test-password"}}
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:9093/login", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "http://127.0.0.1:9092")
	recorder := httptest.NewRecorder()

	s.login(recorder, request)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("login should not reject a browser origin mismatch: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if len(recorder.Result().Cookies()) == 0 {
		t.Fatal("login did not create a session cookie")
	}
}

func TestValidOriginAllowsSameHostAcrossPorts(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:9093/issue", nil)
	request.Header.Set("Origin", "http://127.0.0.1:9092")
	if !validOrigin(request) {
		t.Fatal("same host with a different service port should be accepted")
	}

	request.Header.Set("Origin", "null")
	if !validOrigin(request) {
		t.Fatal("opaque browser origin should rely on the CSRF token")
	}

	request.Header.Set("Origin", "https://attacker.example")
	if validOrigin(request) {
		t.Fatal("a different origin host must be rejected")
	}
}

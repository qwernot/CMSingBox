package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"cmsingbox.local/cmsingbox/internal/licensing"
)

type session struct {
	Expires time.Time
	CSRF    string
}

type server struct {
	privateKey  []byte
	passwordSum [sha256.Size]byte
	auditPath   string
	secure      bool
	mu          sync.Mutex
	sessions    map[string]session
	tmpl        *template.Template
}

type pageData struct {
	Authenticated bool
	CSRF          string
	Error         string
	Token         string
	DeviceCode    string
	Subscriptions int
	Expires       string
	LicenseID     string
}

type auditEntry struct {
	Time             time.Time `json:"time"`
	RemoteAddress    string    `json:"remote_address"`
	LicenseID        string    `json:"license_id"`
	DeviceCode       string    `json:"device_code"`
	MaxSubscriptions int       `json:"max_subscriptions"`
	ExpiresAt        int64     `json:"expires_at,omitempty"`
}

func main() {
	listen := flag.String("listen", "127.0.0.1:9093", "监听地址")
	privatePath := flag.String("private-key", "", "Ed25519 私钥文件")
	passwordHash := flag.String("password-hash", os.Getenv("CMSINGBOX_LICENSE_PASSWORD_HASH"), "管理员密码的 SHA-256 十六进制值")
	auditPath := flag.String("audit", "license-audit.jsonl", "签发审计日志")
	secureCookie := flag.Bool("secure-cookie", false, "仅通过 HTTPS 发送会话 Cookie")
	flag.Parse()

	if *privatePath == "" || strings.TrimSpace(*passwordHash) == "" {
		log.Fatal("必须设置 -private-key 和 -password-hash")
	}
	raw, err := os.ReadFile(*privatePath)
	if err != nil {
		log.Fatal(err)
	}
	privateKey, err := licensing.ParsePrivateKey(string(raw))
	if err != nil {
		log.Fatal(err)
	}
	decodedHash, err := hex.DecodeString(strings.TrimSpace(*passwordHash))
	if err != nil || len(decodedHash) != sha256.Size {
		log.Fatal("password-hash 必须是 SHA-256 十六进制值")
	}
	s := &server{
		privateKey: privateKey, auditPath: *auditPath, secure: *secureCookie,
		sessions: make(map[string]session), tmpl: template.Must(template.New("page").Parse(pageHTML)),
	}
	copy(s.passwordSum[:], decodedHash)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("GET /", s.home)
	mux.HandleFunc("POST /login", s.login)
	mux.HandleFunc("POST /logout", s.logout)
	mux.HandleFunc("POST /issue", s.issue)
	handler := securityHeaders(mux)
	log.Printf("CMSingBox 授权签发后台监听 %s", *listen)
	log.Fatal(http.ListenAndServe(*listen, handler))
}

func (s *server) home(w http.ResponseWriter, r *http.Request) {
	_, sess, ok := s.currentSession(r)
	s.render(w, pageData{Authenticated: ok, CSRF: sess.CSRF, Subscriptions: 1})
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "请求无效", http.StatusBadRequest)
		return
	}
	sum := sha256.Sum256([]byte(r.FormValue("password")))
	if subtle.ConstantTimeCompare(sum[:], s.passwordSum[:]) != 1 {
		time.Sleep(300 * time.Millisecond)
		s.render(w, pageData{Error: "密码错误", Subscriptions: 1})
		return
	}
	token := randomHex(32)
	sess := session{Expires: time.Now().Add(12 * time.Hour), CSRF: randomHex(24)}
	s.mu.Lock()
	s.sessions[token] = sess
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: "cmsingbox_signer", Value: token, Path: "/", HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteStrictMode, MaxAge: 43200})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	token, sess, ok := s.currentSession(r)
	if !ok || !validCSRF(r, sess) {
		http.Error(w, "会话无效", http.StatusForbidden)
		return
	}
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: "cmsingbox_signer", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteStrictMode})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *server) issue(w http.ResponseWriter, r *http.Request) {
	_, sess, ok := s.currentSession(r)
	if !ok || !validCSRF(r, sess) || !validOrigin(r) {
		http.Error(w, "会话无效", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "请求无效", http.StatusBadRequest)
		return
	}
	data := pageData{Authenticated: true, CSRF: sess.CSRF, DeviceCode: strings.TrimSpace(r.FormValue("device_code")), Expires: strings.TrimSpace(r.FormValue("expires")), LicenseID: strings.TrimSpace(r.FormValue("license_id"))}
	data.Subscriptions, _ = strconv.Atoi(r.FormValue("subscriptions"))
	if len(data.DeviceCode) != 6 || strings.Trim(data.DeviceCode, "0123456789") != "" || data.Subscriptions < 1 {
		data.Error = "设备码必须是六位数字，订阅额度必须大于 0"
		s.render(w, data)
		return
	}
	var expiresAt int64
	if data.Expires != "" {
		date, err := time.ParseInLocation("2006-01-02", data.Expires, time.Local)
		if err != nil {
			data.Error = "到期日格式无效"
			s.render(w, data)
			return
		}
		expiresAt = date.Add(24 * time.Hour).Unix()
	}
	if data.LicenseID == "" {
		data.LicenseID = "LIC-" + strings.ToUpper(randomHex(6))
	}
	claims := licensing.Claims{LicenseID: data.LicenseID, DeviceCode: data.DeviceCode, MaxSubscriptions: data.Subscriptions, ExpiresAt: expiresAt}
	token, err := licensing.Issue(s.privateKey, claims)
	if err != nil {
		data.Error = err.Error()
		s.render(w, data)
		return
	}
	data.Token = token
	if err := s.audit(auditEntry{Time: time.Now(), RemoteAddress: remoteIP(r.RemoteAddr), LicenseID: data.LicenseID, DeviceCode: data.DeviceCode, MaxSubscriptions: data.Subscriptions, ExpiresAt: expiresAt}); err != nil {
		log.Printf("写入审计日志失败: %v", err)
	}
	s.render(w, data)
}

func (s *server) currentSession(r *http.Request) (string, session, bool) {
	cookie, err := r.Cookie("cmsingbox_signer")
	if err != nil {
		return "", session{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[cookie.Value]
	if !ok || time.Now().After(sess.Expires) {
		delete(s.sessions, cookie.Value)
		return "", session{}, false
	}
	return cookie.Value, sess, true
}

func (s *server) audit(entry auditEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.auditPath), 0700); err != nil && filepath.Dir(s.auditPath) != "." {
		return err
	}
	f, err := os.OpenFile(s.auditPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(entry)
}

func (s *server) render(w http.ResponseWriter, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.Execute(w, data); err != nil {
		log.Printf("渲染页面失败: %v", err)
	}
}

func validCSRF(r *http.Request, sess session) bool {
	return subtle.ConstantTimeCompare([]byte(r.FormValue("csrf")), []byte(sess.CSRF)) == 1
}

func validOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" || origin == "null" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Hostname() == "" {
		return false
	}
	requestHost := r.Host
	if host, _, splitErr := net.SplitHostPort(r.Host); splitErr == nil {
		requestHost = host
	}
	return strings.EqualFold(u.Hostname(), strings.Trim(requestHost, "[]"))
}

func remoteIP(address string) string {
	if index := strings.LastIndex(address, ":"); index >= 0 {
		return strings.Trim(address[:index], "[]")
	}
	return address
}

func randomHex(bytes int) string {
	raw := make([]byte, bytes)
	if _, err := rand.Read(raw); err != nil {
		panic(err)
	}
	return hex.EncodeToString(raw)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

const pageHTML = `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>CMSingBox 授权中心</title><style>
*{box-sizing:border-box}body{margin:0;background:#07101f;color:#e5edf8;font:15px system-ui,sans-serif}.wrap{max-width:900px;margin:0 auto;padding:48px 20px}.card{background:#101b2e;border:1px solid #24324a;border-radius:18px;padding:26px;box-shadow:0 18px 50px #0005}h1{margin:0 0 6px;font-size:28px}.muted{color:#8fa2bd}.grid{display:grid;grid-template-columns:1fr 1fr;gap:16px}label{display:block;color:#a9b8cb;margin-bottom:7px}input,textarea{width:100%;border:1px solid #33445e;background:#091426;color:#fff;border-radius:10px;padding:12px;font:inherit}textarea{min-height:150px;resize:vertical}.full{grid-column:1/-1}.actions{display:flex;gap:10px;margin-top:20px}button{border:0;border-radius:10px;padding:11px 18px;background:#3277ff;color:white;font-weight:700;cursor:pointer}.secondary{background:#283750}.error{background:#5c1e2a;color:#ffc5cf;padding:12px;border-radius:10px;margin:16px 0}.result{margin-top:24px;padding-top:20px;border-top:1px solid #283750}.login{max-width:420px;margin:12vh auto}.badge{display:inline-block;background:#123c35;color:#73e4bd;padding:5px 9px;border-radius:99px}@media(max-width:650px){.grid{grid-template-columns:1fr}.full{grid-column:auto}}
</style></head><body><div class="wrap">{{if .Authenticated}}<div class="card"><div style="display:flex;justify-content:space-between;gap:16px"><div><h1>CMSingBox 授权中心</h1><p class="muted">离线签发 · Ed25519 · 私钥不离开本服务</p></div><form method="post" action="/logout"><input type="hidden" name="csrf" value="{{.CSRF}}"><button class="secondary">退出</button></form></div>{{if .Error}}<div class="error">{{.Error}}</div>{{end}}<form method="post" action="/issue"><input type="hidden" name="csrf" value="{{.CSRF}}"><div class="grid"><div><label>设备码</label><input name="device_code" value="{{.DeviceCode}}" maxlength="6" pattern="[0-9]{6}" required placeholder="237438"></div><div><label>订阅链接额度</label><input name="subscriptions" value="{{.Subscriptions}}" type="number" min="1" required></div><div><label>到期日（留空永久）</label><input name="expires" value="{{.Expires}}" type="date"></div><div><label>授权编号（可留空）</label><input name="license_id" value="{{.LicenseID}}" placeholder="customer-001"></div></div><div class="actions"><button type="submit">生成授权码</button></div></form>{{if .Token}}<div class="result"><span class="badge">签发成功</span><p>授权编号：<b>{{.LicenseID}}</b></p><textarea id="token" readonly>{{.Token}}</textarea><div class="actions"><button onclick="navigator.clipboard.writeText(document.getElementById('token').value);this.textContent='已复制'" type="button">复制授权码</button></div></div>{{end}}</div>{{else}}<div class="card login"><h1>CMSingBox 授权中心</h1><p class="muted">仅限授权管理员使用</p>{{if .Error}}<div class="error">{{.Error}}</div>{{end}}<form method="post" action="/login"><label>管理员密码</label><input name="password" type="password" autocomplete="current-password" required autofocus><div class="actions"><button type="submit">登录</button></div></form></div>{{end}}</div></body></html>`

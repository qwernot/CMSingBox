package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"cmsingbox.local/cmsingbox/internal/storage"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookieName = "cmsingbox_session"
	sessionLifetime   = 24 * time.Hour
)

func newSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Server) sessionToken(c *gin.Context) string {
	if cookie, err := c.Cookie(sessionCookieName); err == nil {
		return cookie
	}
	if value := c.GetHeader("Authorization"); strings.HasPrefix(value, "Bearer ") {
		return strings.TrimPrefix(value, "Bearer ")
	}
	return ""
}

func (s *Server) validSession(token string) bool {
	if token == "" {
		return false
	}
	s.sessionsMu.RLock()
	expires, ok := s.sessions[token]
	s.sessionsMu.RUnlock()
	if !ok || time.Now().After(expires) {
		if ok {
			s.sessionsMu.Lock()
			delete(s.sessions, token)
			s.sessionsMu.Unlock()
		}
		return false
	}
	return true
}

func (s *Server) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.validSession(s.sessionToken(c)) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
			return
		}
		c.Next()
	}
}

func (s *Server) login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码不能为空"})
		return
	}
	auth := s.store.GetAuthConfig()
	if req.Username != auth.Username || bcrypt.CompareHashAndPassword([]byte(auth.PasswordHash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	token, err := newSessionToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建会话失败"})
		return
	}
	expires := time.Now().Add(sessionLifetime)
	s.sessionsMu.Lock()
	s.sessions[token] = expires
	s.sessionsMu.Unlock()
	http.SetCookie(c.Writer, &http.Cookie{Name: sessionCookieName, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: int(sessionLifetime.Seconds())})
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"username": auth.Username, "expires_at": expires}})
}

func (s *Server) authStatus(c *gin.Context) {
	if !s.validSession(s.sessionToken(c)) {
		c.JSON(http.StatusUnauthorized, gin.H{"data": gin.H{"authenticated": false}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"authenticated": true, "username": s.store.GetAuthConfig().Username}})
}

func (s *Server) logout(c *gin.Context) {
	token := s.sessionToken(c)
	s.sessionsMu.Lock()
	delete(s.sessions, token)
	s.sessionsMu.Unlock()
	http.SetCookie(c.Writer, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: -1})
	c.JSON(http.StatusOK, gin.H{"message": "已退出登录"})
}

func (s *Server) changePassword(c *gin.Context) {
	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "新密码至少需要 8 个字符"})
		return
	}
	auth := s.store.GetAuthConfig()
	if bcrypt.CompareHashAndPassword([]byte(auth.PasswordHash), []byte(req.CurrentPassword)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "当前密码错误"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成密码哈希失败"})
		return
	}
	if err := s.store.UpdateAuthConfig(storage.AuthConfig{Username: auth.Username, PasswordHash: string(hash)}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 修改密码后仅保留当前会话。
	current := s.sessionToken(c)
	s.sessionsMu.Lock()
	expires := s.sessions[current]
	s.sessions = map[string]time.Time{current: expires}
	s.sessionsMu.Unlock()
	c.JSON(http.StatusOK, gin.H{"message": "密码已更新"})
}

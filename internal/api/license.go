package api

import (
	"errors"
	"net/http"

	"cmsingbox.local/cmsingbox/internal/licensing"
	"github.com/gin-gonic/gin"
)

func (s *Server) licenseStatus(c *gin.Context) {
	if s.license == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "授权模块未初始化"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": s.licenseResponse()})
}

func (s *Server) activateLicense(c *gin.Context) {
	if s.license == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "授权模块未初始化"})
		return
	}
	var req struct {
		LicenseCode string `json:"license_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入授权码"})
		return
	}
	status, err := s.license.Activate(req.LicenseCode)
	if err != nil {
		code := http.StatusBadRequest
		if errors.Is(err, licensing.ErrNotConfigured) {
			code = http.StatusServiceUnavailable
		}
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}
	s.store.SetSubscriptionLimit(status.MaxSubscriptions)
	c.JSON(http.StatusOK, gin.H{"data": s.licenseResponse(), "message": "授权成功"})
}

func (s *Server) clearLicense(c *gin.Context) {
	if s.license == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "授权模块未初始化"})
		return
	}
	if err := s.license.Clear(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.store.SetSubscriptionLimit(s.license.Limit())
	c.JSON(http.StatusOK, gin.H{"data": s.licenseResponse(), "message": "授权已清除"})
}

func (s *Server) licenseResponse() gin.H {
	status := s.license.Status()
	s.store.SetSubscriptionLimit(status.MaxSubscriptions)
	used, limit := s.store.SubscriptionUsage()
	remaining := limit - used
	if remaining < 0 {
		remaining = 0
	}
	return gin.H{
		"is_valid": status.Valid, "status": status.Status, "reason": status.Reason,
		"device_code": status.DeviceCode, "license_id": status.LicenseID,
		"max_subscriptions": limit, "used_subscriptions": used,
		"remaining_subscriptions": remaining, "expires_at": status.ExpiresAt,
	}
}

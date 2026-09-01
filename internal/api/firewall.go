package api

import (
	"net/http"

	"cmsingbox.local/cmsingbox/internal/firewall"
	"github.com/gin-gonic/gin"
)

func (s *Server) firewallConfig() firewall.Config {
	settings := s.store.GetSettings()
	return firewall.Config{Port: settings.TProxyPort, BypassCIDRs: settings.BypassCIDRs}
}

func (s *Server) getFirewallStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"supported": s.firewall.Supported(), "active": s.firewall.Active()}})
}

func (s *Server) previewFirewall(c *gin.Context) {
	script, err := firewall.BuildNFTables(s.firewallConfig())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": script})
}

func (s *Server) applyFirewall(c *gin.Context) {
	if !s.store.GetSettings().TransparentProxy {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先在设置中启用透明代理"})
		return
	}
	if err := s.firewall.Apply(s.firewallConfig()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "透明代理规则已应用"})
}

func (s *Server) disableFirewall(c *gin.Context) {
	if err := s.firewall.Disable(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "透明代理规则已停用"})
}

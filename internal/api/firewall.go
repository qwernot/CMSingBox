package api

import (
	"fmt"
	"net/http"

	"cmsingbox.local/cmsingbox/internal/firewall"
	"github.com/gin-gonic/gin"
)

func (s *Server) firewallConfig() firewall.Config {
	settings := s.store.GetSettings()
	return firewall.Config{Port: settings.TProxyPort, BypassCIDRs: settings.BypassCIDRs, Backend: settings.TransparentBackend}
}

func (s *Server) getFirewallStatus(c *gin.Context) {
	config := s.firewallConfig()
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"supported": s.firewall.SupportedFor(config), "active": s.firewall.ActiveFor(config)}})
}

func (s *Server) previewFirewall(c *gin.Context) {
	config := s.firewallConfig()
	if config.Backend == "routeros_redirect" {
		c.JSON(http.StatusOK, gin.H{"data": fmt.Sprintf("iptables-legacy -t nat -N CMSINGBOX_REDIRECT\niptables-legacy -t nat -A CMSINGBOX_REDIRECT -p tcp -j REDIRECT --to-ports %d\niptables-legacy -t nat -A PREROUTING -p tcp -j CMSINGBOX_REDIRECT\n", config.Port)})
		return
	}
	script, err := firewall.BuildNFTables(config)
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
	if err := s.firewall.DisableFor(s.firewallConfig()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "透明代理规则已停用"})
}

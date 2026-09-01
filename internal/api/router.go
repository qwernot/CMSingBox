package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io/fs"
	stdnet "net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"cmsingbox.local/cmsingbox/internal/builder"
	"cmsingbox.local/cmsingbox/internal/daemon"
	"cmsingbox.local/cmsingbox/internal/dnsproxy"
	"cmsingbox.local/cmsingbox/internal/firewall"
	"cmsingbox.local/cmsingbox/internal/kernel"
	"cmsingbox.local/cmsingbox/internal/licensing"
	"cmsingbox.local/cmsingbox/internal/logger"
	"cmsingbox.local/cmsingbox/internal/maintenance"
	"cmsingbox.local/cmsingbox/internal/parser"
	"cmsingbox.local/cmsingbox/internal/service"
	"cmsingbox.local/cmsingbox/internal/storage"
	"cmsingbox.local/cmsingbox/web"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	gnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// generateRandomSecret 生成随机密钥
func generateRandomSecret(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// 如果加密随机数生成失败，返回空字符串
		return ""
	}
	return hex.EncodeToString(bytes)[:length]
}

// Server API 服务器
type Server struct {
	store          *storage.JSONStore
	subService     *service.SubscriptionService
	processManager *daemon.ProcessManager
	launchdManager *daemon.LaunchdManager
	systemdManager *daemon.SystemdManager
	kernelManager  *kernel.Manager
	scheduler      *service.Scheduler
	dnsService     *dnsproxy.Service
	firewall       *firewall.Manager
	cleaner        *maintenance.Cleaner
	router         *gin.Engine
	sbmPath        string // sbm 可执行文件路径
	port           int    // Web 服务端口
	version        string // sbm 版本号
	sessions       map[string]time.Time
	sessionsMu     sync.RWMutex
	networkMu      sync.Mutex
	lastNetSent    uint64
	lastNetRecv    uint64
	lastNetAt      time.Time
	license        *licensing.Manager
}

// NewServer 创建 API 服务器
func NewServer(store *storage.JSONStore, processManager *daemon.ProcessManager, launchdManager *daemon.LaunchdManager, systemdManager *daemon.SystemdManager, sbmPath string, port int, version string, license *licensing.Manager) *Server {
	gin.SetMode(gin.ReleaseMode)

	subService := service.NewSubscriptionService(store)

	// 创建内核管理器
	kernelManager := kernel.NewManager(store.GetDataDir(), store.GetSettings)

	dnsService := dnsproxy.New(dnsConfig(store.GetSettings()))
	s := &Server{
		store:          store,
		subService:     subService,
		processManager: processManager,
		launchdManager: launchdManager,
		systemdManager: systemdManager,
		kernelManager:  kernelManager,
		scheduler:      service.NewScheduler(store, subService),
		dnsService:     dnsService,
		firewall:       firewall.New(),
		cleaner:        maintenance.New(store.GetDataDir()),
		router:         gin.Default(),
		sbmPath:        sbmPath,
		port:           port,
		version:        version,
		sessions:       make(map[string]time.Time),
		license:        license,
	}

	// 设置调度器的更新回调
	s.scheduler.SetUpdateCallback(s.autoApplyConfig)

	s.setupRoutes()
	if err := dnsService.Start(); err != nil {
		logger.Printf("启动 DNS 服务失败: %v", err)
	}
	return s
}

func dnsConfig(settings *storage.Settings) dnsproxy.Config {
	return dnsproxy.Config{Enabled: settings.DNSEnabled, Listen: settings.DNSListen, ProxyUpstream: settings.DNSProxyUpstream, DirectUpstream: settings.DNSDirectUpstream, Mode: settings.DNSRoutingMode, Exceptions: settings.DNSExceptions}
}

// StartScheduler 启动定时任务调度器
func (s *Server) StartScheduler() {
	s.scheduler.Start()
}

// StopScheduler 停止定时任务调度器
func (s *Server) StopScheduler() {
	s.scheduler.Stop()
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// CORS 配置
	s.router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	// API 路由组
	api := s.router.Group("/api")
	{
		api.POST("/auth/login", s.login)
		api.GET("/auth/status", s.authStatus)

		protected := api.Group("")
		protected.Use(s.requireAuth())
		protected.POST("/auth/logout", s.logout)
		protected.PUT("/auth/password", s.changePassword)
		protected.GET("/backup", s.exportBackup)
		protected.POST("/backup/import", s.importBackup)
		protected.GET("/license/status", s.licenseStatus)
		protected.POST("/license/activate", s.activateLicense)
		protected.POST("/license/clear", s.clearLicense)

		// 订阅管理
		protected.GET("/subscriptions", s.getSubscriptions)
		protected.POST("/subscriptions", s.addSubscription)
		protected.PUT("/subscriptions/:id", s.updateSubscription)
		protected.DELETE("/subscriptions/:id", s.deleteSubscription)
		protected.POST("/subscriptions/:id/refresh", s.refreshSubscription)
		protected.POST("/subscriptions/refresh-all", s.refreshAllSubscriptions)

		// 过滤器管理
		protected.GET("/filters", s.getFilters)
		protected.POST("/filters", s.addFilter)
		protected.PUT("/filters/:id", s.updateFilter)
		protected.DELETE("/filters/:id", s.deleteFilter)

		// 规则管理
		protected.GET("/rules", s.getRules)
		protected.POST("/rules", s.addRule)
		protected.PUT("/rules/:id", s.updateRule)
		protected.DELETE("/rules/:id", s.deleteRule)

		// 规则组管理
		protected.GET("/rule-groups", s.getRuleGroups)
		protected.PUT("/rule-groups/:id", s.updateRuleGroup)

		// 规则集验证
		protected.GET("/ruleset/validate", s.validateRuleSet)

		// 设置
		protected.GET("/settings", s.getSettings)
		protected.PUT("/settings", s.updateSettings)

		// 系统 hosts
		protected.GET("/system-hosts", s.getSystemHosts)

		// 配置生成
		protected.POST("/config/generate", s.generateConfig)
		protected.POST("/config/apply", s.applyConfig)
		protected.GET("/config/preview", s.previewConfig)

		// 服务管理
		protected.GET("/service/status", s.getServiceStatus)
		protected.POST("/service/start", s.startService)
		protected.POST("/service/stop", s.stopService)
		protected.POST("/service/restart", s.restartService)
		protected.POST("/service/reload", s.reloadService)

		// launchd 管理
		protected.GET("/launchd/status", s.getLaunchdStatus)
		protected.POST("/launchd/install", s.installLaunchd)
		protected.POST("/launchd/uninstall", s.uninstallLaunchd)
		protected.POST("/launchd/restart", s.restartLaunchd)

		// systemd 管理
		protected.GET("/systemd/status", s.getSystemdStatus)
		protected.POST("/systemd/install", s.installSystemd)
		protected.POST("/systemd/uninstall", s.uninstallSystemd)
		protected.POST("/systemd/restart", s.restartSystemd)

		// 统一守护进程管理（自动判断系统）
		protected.GET("/daemon/status", s.getDaemonStatus)
		protected.POST("/daemon/install", s.installDaemon)
		protected.POST("/daemon/uninstall", s.uninstallDaemon)
		protected.POST("/daemon/restart", s.restartDaemon)

		// 系统监控
		protected.GET("/monitor/system", s.getSystemInfo)
		protected.GET("/monitor/logs", s.getLogs)
		protected.GET("/monitor/logs/sbm", s.getAppLogs)
		protected.GET("/monitor/logs/singbox", s.getSingboxLogs)
		protected.GET("/monitor/dns", s.getDNSMonitor)
		protected.GET("/firewall/status", s.getFirewallStatus)
		protected.GET("/firewall/preview", s.previewFirewall)
		protected.POST("/firewall/apply", s.applyFirewall)
		protected.POST("/firewall/disable", s.disableFirewall)
		protected.GET("/system/cleanup", s.previewCleanup)
		protected.POST("/system/cleanup", s.runCleanup)

		// 节点
		protected.GET("/nodes", s.getAllNodes)
		protected.GET("/nodes/countries", s.getCountryGroups)
		protected.GET("/nodes/country/:code", s.getNodesByCountry)
		protected.POST("/nodes/parse", s.parseNodeURL)

		// 手动节点
		protected.GET("/manual-nodes", s.getManualNodes)
		protected.POST("/manual-nodes", s.addManualNode)
		protected.PUT("/manual-nodes/:id", s.updateManualNode)
		protected.DELETE("/manual-nodes/:id", s.deleteManualNode)

		// 内核管理
		protected.GET("/kernel/info", s.getKernelInfo)
		protected.GET("/kernel/releases", s.getKernelReleases)
		protected.POST("/kernel/download", s.startKernelDownload)
		protected.GET("/kernel/progress", s.getKernelProgress)
	}
	s.router.GET("/client/:path", s.clientConfig)

	// 静态文件服务（前端，使用嵌入的文件系统）
	distFS, err := web.GetDistFS()
	if err != nil {
		logger.Printf("加载前端资源失败: %v", err)
	} else {
		// 获取 assets 子目录
		assetsFS, _ := fs.Sub(distFS, "assets")
		s.router.StaticFS("/assets", http.FS(assetsFS))

		// 处理根路径和所有未匹配的路由（SPA 支持）
		indexHTML, _ := fs.ReadFile(distFS, "index.html")
		s.router.GET("/", func(c *gin.Context) {
			c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
		})
		s.router.NoRoute(func(c *gin.Context) {
			c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
		})
	}
}

// Run 运行服务器
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

// ==================== 订阅 API ====================

func (s *Server) getSubscriptions(c *gin.Context) {
	subs := s.subService.GetAll()
	c.JSON(http.StatusOK, gin.H{"data": subs})
}

func (s *Server) addSubscription(c *gin.Context) {
	if s.license != nil {
		s.store.SetSubscriptionLimit(s.license.Limit())
	}
	var req struct {
		Name string `json:"name" binding:"required"`
		URL  string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub, err := s.subService.Add(req.Name, req.URL)
	if err != nil {
		if errors.Is(err, storage.ErrSubscriptionLimitExceeded) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 自动应用配置
	if err := s.autoApplyConfig(); err != nil {
		c.JSON(http.StatusOK, gin.H{"data": sub, "warning": "添加成功，但自动应用配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": sub})
}

func (s *Server) updateSubscription(c *gin.Context) {
	id := c.Param("id")

	var sub storage.Subscription
	if err := c.ShouldBindJSON(&sub); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub.ID = id
	if err := s.subService.Update(sub); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 自动应用配置
	if err := s.autoApplyConfig(); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "更新成功，但自动应用配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

func (s *Server) deleteSubscription(c *gin.Context) {
	id := c.Param("id")

	if err := s.subService.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 自动应用配置
	if err := s.autoApplyConfig(); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "删除成功，但自动应用配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func (s *Server) refreshSubscription(c *gin.Context) {
	id := c.Param("id")

	if err := s.subService.Refresh(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 自动应用配置
	if err := s.autoApplyConfig(); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "刷新成功，但自动应用配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "刷新成功"})
}

func (s *Server) refreshAllSubscriptions(c *gin.Context) {
	if err := s.subService.RefreshAll(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 自动应用配置
	if err := s.autoApplyConfig(); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "刷新成功，但自动应用配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "刷新成功"})
}

// ==================== 过滤器 API ====================

func (s *Server) getFilters(c *gin.Context) {
	filters := s.store.GetFilters()
	c.JSON(http.StatusOK, gin.H{"data": filters})
}

func (s *Server) addFilter(c *gin.Context) {
	var filter storage.Filter
	if err := c.ShouldBindJSON(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 生成 ID
	filter.ID = uuid.New().String()

	if err := s.store.AddFilter(filter); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 自动应用配置
	if err := s.autoApplyConfig(); err != nil {
		c.JSON(http.StatusOK, gin.H{"data": filter, "warning": "添加成功，但自动应用配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": filter})
}

func (s *Server) updateFilter(c *gin.Context) {
	id := c.Param("id")

	var filter storage.Filter
	if err := c.ShouldBindJSON(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter.ID = id
	if err := s.store.UpdateFilter(filter); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 自动应用配置
	if err := s.autoApplyConfig(); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "更新成功，但自动应用配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

func (s *Server) deleteFilter(c *gin.Context) {
	id := c.Param("id")

	if err := s.store.DeleteFilter(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 自动应用配置
	if err := s.autoApplyConfig(); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "删除成功，但自动应用配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ==================== 规则 API ====================

func (s *Server) getRules(c *gin.Context) {
	rules := s.store.GetRules()
	c.JSON(http.StatusOK, gin.H{"data": rules})
}

func (s *Server) addRule(c *gin.Context) {
	var rule storage.Rule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 生成 ID
	rule.ID = uuid.New().String()

	if err := s.store.AddRule(rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 自动应用配置
	if err := s.autoApplyConfig(); err != nil {
		c.JSON(http.StatusOK, gin.H{"data": rule, "warning": "添加成功，但自动应用配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": rule})
}

func (s *Server) updateRule(c *gin.Context) {
	id := c.Param("id")

	var rule storage.Rule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule.ID = id
	if err := s.store.UpdateRule(rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 自动应用配置
	if err := s.autoApplyConfig(); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "更新成功，但自动应用配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

func (s *Server) deleteRule(c *gin.Context) {
	id := c.Param("id")

	if err := s.store.DeleteRule(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 自动应用配置
	if err := s.autoApplyConfig(); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "删除成功，但自动应用配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ==================== 规则组 API ====================

func (s *Server) getRuleGroups(c *gin.Context) {
	ruleGroups := s.store.GetRuleGroups()
	c.JSON(http.StatusOK, gin.H{"data": ruleGroups})
}

func (s *Server) updateRuleGroup(c *gin.Context) {
	id := c.Param("id")

	var ruleGroup storage.RuleGroup
	if err := c.ShouldBindJSON(&ruleGroup); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ruleGroup.ID = id
	if err := s.store.UpdateRuleGroup(ruleGroup); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 自动应用配置
	if err := s.autoApplyConfig(); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "更新成功，但自动应用配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// ==================== 规则集验证 API ====================

func (s *Server) validateRuleSet(c *gin.Context) {
	ruleType := c.Query("type") // geosite 或 geoip
	name := c.Query("name")     // 规则集名称

	if ruleType == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数 type 和 name 是必需的"})
		return
	}

	if ruleType != "geosite" && ruleType != "geoip" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type 必须是 geosite 或 geoip"})
		return
	}

	settings := s.store.GetSettings()
	var url string
	var tag string

	if ruleType == "geosite" {
		tag = "geosite-" + name
		url = settings.RuleSetBaseURL + "/geosite-" + name + ".srs"
	} else {
		tag = "geoip-" + name
		// geoip 使用相对路径
		url = settings.RuleSetBaseURL + "/../rule-set-geoip/geoip-" + name + ".srs"
	}

	// 如果配置了 GitHub 代理，添加代理前缀
	if settings.GithubProxy != "" {
		url = settings.GithubProxy + url
	}

	// 发送 HEAD 请求检查文件是否存在
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Head(url)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"valid":   false,
			"url":     url,
			"tag":     tag,
			"message": "无法访问规则集: " + err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		c.JSON(http.StatusOK, gin.H{
			"valid":   true,
			"url":     url,
			"tag":     tag,
			"message": "规则集存在",
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"valid":   false,
			"url":     url,
			"tag":     tag,
			"message": "规则集不存在 (HTTP " + strconv.Itoa(resp.StatusCode) + ")",
		})
	}
}

// ==================== 设置 API ====================

func (s *Server) getSettings(c *gin.Context) {
	settings := s.store.GetSettings()
	settings.WebPort = s.port
	c.JSON(http.StatusOK, gin.H{"data": settings})
}

func (s *Server) updateSettings(c *gin.Context) {
	var settings storage.Settings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 根据局域网访问设置处理 secret
	if settings.AllowLAN {
		if settings.MixedAuthEnabled {
			if settings.MixedUsername == "" {
				settings.MixedUsername = "cmsingbox"
			}
			if settings.MixedPassword == "" {
				settings.MixedPassword = generateRandomSecret(24)
			}
		}
		// 开启局域网访问且 secret 为空时，自动生成一个
		if settings.ClashAPISecret == "" {
			settings.ClashAPISecret = generateRandomSecret(16)
		}
	} else {
		// 关闭局域网访问时，清除 secret
		settings.ClashAPISecret = ""
	}

	if err := s.store.UpdateSettings(&settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := s.dnsService.Update(dnsConfig(&settings)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DNS 配置已保存但服务启动失败: " + err.Error()})
		return
	}

	// 更新进程管理器的配置路径（sing-box 路径是固定的，无需更新）
	s.processManager.SetConfigPath(s.resolvePath(settings.ConfigPath))

	// 重启调度器（可能更新了定时间隔）
	s.scheduler.Restart()

	// 自动应用配置
	if err := s.autoApplyConfig(); err != nil {
		c.JSON(http.StatusOK, gin.H{"data": settings, "warning": "更新成功，但自动应用配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": settings, "message": "更新成功"})
}

// ==================== 系统 hosts API ====================

func (s *Server) getSystemHosts(c *gin.Context) {
	hosts := builder.ParseSystemHosts()

	var entries []storage.HostEntry
	for domain, ips := range hosts {
		entries = append(entries, storage.HostEntry{
			ID:      "system-" + domain,
			Domain:  domain,
			IPs:     ips,
			Enabled: true,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": entries})
}

// ==================== 配置 API ====================

func (s *Server) generateConfig(c *gin.Context) {
	configJSON, err := s.buildConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": configJSON})
}

func (s *Server) previewConfig(c *gin.Context) {
	configJSON, err := s.buildConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.String(http.StatusOK, configJSON)
}

func (s *Server) applyConfig(c *gin.Context) {
	configJSON, err := s.buildConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 保存配置文件
	settings := s.store.GetSettings()
	if err := s.saveConfigFile(s.resolvePath(settings.ConfigPath), configJSON); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 检查配置
	if err := s.processManager.Check(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 重启服务
	if s.processManager.IsRunning() {
		if err := s.processManager.Restart(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "配置已应用"})
}

func (s *Server) buildConfig() (string, error) {
	settings := s.store.GetSettings()
	nodes := s.store.GetAllNodes()
	filters := s.store.GetFilters()
	rules := s.store.GetRules()
	ruleGroups := s.store.GetRuleGroups()

	b := builder.NewConfigBuilder(settings, nodes, filters, rules, ruleGroups)
	if s.processManager != nil {
		if version, err := s.processManager.Version(); err == nil {
			b = b.WithSingBoxVersion(version)
		}
	}
	return b.BuildJSON()
}

func (s *Server) saveConfigFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// resolvePath 将相对路径解析为基于数据目录的绝对路径
func (s *Server) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(s.store.GetDataDir(), path)
}

// autoApplyConfig 自动应用配置（如果 sing-box 正在运行）
func (s *Server) autoApplyConfig() error {
	settings := s.store.GetSettings()
	if !settings.AutoApply {
		return nil
	}

	// 生成配置
	configJSON, err := s.buildConfig()
	if err != nil {
		return err
	}

	// 保存配置文件
	if err := s.saveConfigFile(s.resolvePath(settings.ConfigPath), configJSON); err != nil {
		return err
	}

	// 如果 sing-box 正在运行，则重启
	if s.processManager.IsRunning() {
		return s.processManager.Restart()
	}

	return nil
}

// ==================== 服务 API ====================

func (s *Server) getServiceStatus(c *gin.Context) {
	running := s.processManager.IsRunning()
	pid := s.processManager.GetPID()

	version := ""
	if v, err := s.processManager.Version(); err == nil {
		version = v
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"running":     running,
			"pid":         pid,
			"version":     version,
			"sbm_version": s.version,
		},
	})
}

func (s *Server) startService(c *gin.Context) {
	if err := s.processManager.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "服务已启动"})
}

func (s *Server) stopService(c *gin.Context) {
	if err := s.processManager.Stop(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "服务已停止"})
}

func (s *Server) restartService(c *gin.Context) {
	if err := s.processManager.Restart(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "服务已重启"})
}

func (s *Server) reloadService(c *gin.Context) {
	if err := s.processManager.Reload(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "配置已重载"})
}

// ==================== launchd API ====================

func (s *Server) getLaunchdStatus(c *gin.Context) {
	if s.launchdManager == nil {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"installed": false,
				"running":   false,
				"plistPath": "",
				"supported": false,
			},
		})
		return
	}

	installed := s.launchdManager.IsInstalled()
	running := s.launchdManager.IsRunning()

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"installed": installed,
			"running":   running,
			"plistPath": s.launchdManager.GetPlistPath(),
			"supported": true,
		},
	})
}

func (s *Server) installLaunchd(c *gin.Context) {
	if s.launchdManager == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前系统不支持 launchd 服务"})
		return
	}

	// 获取用户主目录（支持多种方式）
	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		// 备用方案：使用 os/user 包
		if u, err := user.Current(); err == nil && u.HomeDir != "" {
			homeDir = u.HomeDir
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户目录失败"})
			return
		}
	}

	// 确保日志目录存在
	logsDir := s.store.GetDataDir() + "/logs"

	config := daemon.LaunchdConfig{
		SbmPath:    s.sbmPath,
		DataDir:    s.store.GetDataDir(),
		Port:       strconv.Itoa(s.port),
		LogPath:    logsDir,
		WorkingDir: s.store.GetDataDir(),
		HomeDir:    homeDir,
		RunAtLoad:  true,
		KeepAlive:  true,
	}

	if err := s.launchdManager.Install(config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 安装成功后启动服务
	if err := s.launchdManager.Start(); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "服务已安装，但启动失败: " + err.Error() + "。请重启电脑或手动执行 launchctl load 命令",
			"action":  "manual",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "服务已安装并启动，您可以关闭此终端窗口。sbm 将在后台运行并开机自启。",
		"action":  "exit",
	})
}

func (s *Server) uninstallLaunchd(c *gin.Context) {
	if s.launchdManager == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前系统不支持 launchd 服务"})
		return
	}

	if err := s.launchdManager.Uninstall(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "服务已卸载"})
}

func (s *Server) restartLaunchd(c *gin.Context) {
	if s.launchdManager == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前系统不支持 launchd 服务"})
		return
	}

	if err := s.launchdManager.Restart(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "服务已重启"})
}

// ==================== systemd API ====================

func (s *Server) getSystemdStatus(c *gin.Context) {
	if s.systemdManager == nil {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"installed":   false,
				"running":     false,
				"servicePath": "",
				"supported":   false,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"installed":   s.systemdManager.IsInstalled(),
			"running":     s.systemdManager.IsRunning(),
			"servicePath": s.systemdManager.GetServicePath(),
			"supported":   true,
		},
	})
}

func (s *Server) installSystemd(c *gin.Context) {
	if s.systemdManager == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前系统不支持 systemd 服务"})
		return
	}

	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		if u, err := user.Current(); err == nil && u.HomeDir != "" {
			homeDir = u.HomeDir
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户目录失败"})
			return
		}
	}

	logsDir := s.store.GetDataDir() + "/logs"

	config := daemon.SystemdConfig{
		SbmPath:    s.sbmPath,
		DataDir:    s.store.GetDataDir(),
		Port:       strconv.Itoa(s.port),
		LogPath:    logsDir,
		WorkingDir: s.store.GetDataDir(),
		HomeDir:    homeDir,
		RunAtLoad:  true,
		KeepAlive:  true,
	}

	if err := s.systemdManager.Install(config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := s.systemdManager.Start(); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "服务已安装，但启动失败: " + err.Error() + "。请执行 systemctl --user start cmsingbox",
			"action":  "manual",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "服务已安装并启动，您可以关闭此终端窗口。sbm 将在后台运行并开机自启。",
		"action":  "exit",
	})
}

func (s *Server) uninstallSystemd(c *gin.Context) {
	if s.systemdManager == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前系统不支持 systemd 服务"})
		return
	}

	if err := s.systemdManager.Uninstall(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "服务已卸载"})
}

func (s *Server) restartSystemd(c *gin.Context) {
	if s.systemdManager == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前系统不支持 systemd 服务"})
		return
	}

	if err := s.systemdManager.Restart(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "服务已重启"})
}

// ==================== 统一守护进程 API ====================

func (s *Server) getDaemonStatus(c *gin.Context) {
	platform := runtime.GOOS
	var installed, running, supported bool
	var configPath string

	switch platform {
	case "darwin":
		if s.launchdManager != nil {
			supported = true
			installed = s.launchdManager.IsInstalled()
			running = s.launchdManager.IsRunning()
			configPath = s.launchdManager.GetPlistPath()
		}
	case "linux":
		if s.systemdManager != nil {
			supported = true
			installed = s.systemdManager.IsInstalled()
			running = s.systemdManager.IsRunning()
			configPath = s.systemdManager.GetServicePath()
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"installed":  installed,
			"running":    running,
			"configPath": configPath,
			"supported":  supported,
			"platform":   platform,
		},
	})
}

func (s *Server) installDaemon(c *gin.Context) {
	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		if u, err := user.Current(); err == nil && u.HomeDir != "" {
			homeDir = u.HomeDir
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户目录失败"})
			return
		}
	}

	logsDir := s.store.GetDataDir() + "/logs"

	switch runtime.GOOS {
	case "darwin":
		if s.launchdManager == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前系统不支持守护进程服务"})
			return
		}
		config := daemon.LaunchdConfig{
			SbmPath:    s.sbmPath,
			DataDir:    s.store.GetDataDir(),
			Port:       strconv.Itoa(s.port),
			LogPath:    logsDir,
			WorkingDir: s.store.GetDataDir(),
			HomeDir:    homeDir,
			RunAtLoad:  true,
			KeepAlive:  true,
		}
		if err := s.launchdManager.Install(config); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := s.launchdManager.Start(); err != nil {
			c.JSON(http.StatusOK, gin.H{"message": "服务已安装，但启动失败: " + err.Error(), "action": "manual"})
			return
		}
	case "linux":
		if s.systemdManager == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前系统不支持守护进程服务"})
			return
		}
		config := daemon.SystemdConfig{
			SbmPath:    s.sbmPath,
			DataDir:    s.store.GetDataDir(),
			Port:       strconv.Itoa(s.port),
			LogPath:    logsDir,
			WorkingDir: s.store.GetDataDir(),
			HomeDir:    homeDir,
			RunAtLoad:  true,
			KeepAlive:  true,
		}
		if err := s.systemdManager.Install(config); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := s.systemdManager.Start(); err != nil {
			c.JSON(http.StatusOK, gin.H{"message": "服务已安装，但启动失败: " + err.Error(), "action": "manual"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前系统不支持守护进程服务"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "服务已安装并启动", "action": "exit"})
}

func (s *Server) uninstallDaemon(c *gin.Context) {
	var err error
	switch runtime.GOOS {
	case "darwin":
		if s.launchdManager == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前系统不支持守护进程服务"})
			return
		}
		err = s.launchdManager.Uninstall()
	case "linux":
		if s.systemdManager == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前系统不支持守护进程服务"})
			return
		}
		err = s.systemdManager.Uninstall()
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前系统不支持守护进程服务"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "服务已卸载"})
}

func (s *Server) restartDaemon(c *gin.Context) {
	var err error
	switch runtime.GOOS {
	case "darwin":
		if s.launchdManager == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前系统不支持守护进程服务"})
			return
		}
		err = s.launchdManager.Restart()
	case "linux":
		if s.systemdManager == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前系统不支持守护进程服务"})
			return
		}
		err = s.systemdManager.Restart()
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前系统不支持守护进程服务"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "服务已重启"})
}

// ==================== 监控 API ====================

// ProcessStats 进程资源统计
type ProcessStats struct {
	PID        int     `json:"pid"`
	CPUPercent float64 `json:"cpu_percent"`
	MemoryMB   float64 `json:"memory_mb"`
}

type HostStats struct {
	Hostname       string  `json:"hostname"`
	IPAddress      string  `json:"ip_address"`
	OS             string  `json:"os"`
	Platform       string  `json:"platform"`
	Architecture   string  `json:"architecture"`
	Uptime         uint64  `json:"uptime"`
	CPUModel       string  `json:"cpu_model"`
	CPUPercent     float64 `json:"cpu_percent"`
	MemoryPercent  float64 `json:"memory_percent"`
	MemoryTotal    uint64  `json:"memory_total"`
	DiskPercent    float64 `json:"disk_percent"`
	DiskTotal      uint64  `json:"disk_total"`
	NetworkUpBPS   float64 `json:"network_up_bps"`
	NetworkDownBPS float64 `json:"network_down_bps"`
}

func primaryIPAddress() string {
	interfaces, _ := stdnet.Interfaces()
	for _, iface := range interfaces {
		if iface.Flags&stdnet.FlagUp == 0 || iface.Flags&stdnet.FlagLoopback != 0 {
			continue
		}
		addresses, _ := iface.Addrs()
		for _, address := range addresses {
			ip, _, err := stdnet.ParseCIDR(address.String())
			if err == nil && ip.To4() != nil {
				return ip.String()
			}
		}
	}
	return ""
}

func (s *Server) getSystemInfo(c *gin.Context) {
	result := gin.H{}
	hostStats := HostStats{Architecture: runtime.GOARCH, OS: runtime.GOOS, IPAddress: primaryIPAddress()}
	if info, err := host.Info(); err == nil {
		hostStats.Hostname, hostStats.Platform, hostStats.Uptime = info.Hostname, info.Platform, info.Uptime
	}
	if details, err := cpu.Info(); err == nil && len(details) > 0 {
		hostStats.CPUModel = details[0].ModelName
	}
	if values, err := cpu.Percent(0, false); err == nil && len(values) > 0 {
		hostStats.CPUPercent = values[0]
	}
	if memory, err := mem.VirtualMemory(); err == nil {
		hostStats.MemoryPercent, hostStats.MemoryTotal = memory.UsedPercent, memory.Total
	}
	if usage, err := disk.Usage("/"); err == nil {
		hostStats.DiskPercent, hostStats.DiskTotal = usage.UsedPercent, usage.Total
	}
	if counters, err := gnet.IOCounters(false); err == nil && len(counters) > 0 {
		now := time.Now()
		s.networkMu.Lock()
		if !s.lastNetAt.IsZero() {
			seconds := now.Sub(s.lastNetAt).Seconds()
			if seconds > 0 && counters[0].BytesSent >= s.lastNetSent && counters[0].BytesRecv >= s.lastNetRecv {
				hostStats.NetworkUpBPS = float64(counters[0].BytesSent-s.lastNetSent) / seconds
				hostStats.NetworkDownBPS = float64(counters[0].BytesRecv-s.lastNetRecv) / seconds
			}
		}
		s.lastNetSent, s.lastNetRecv, s.lastNetAt = counters[0].BytesSent, counters[0].BytesRecv, now
		s.networkMu.Unlock()
	}
	result["host"] = hostStats

	// 获取 sbm 进程信息
	sbmPid := int32(os.Getpid())
	if sbmProc, err := process.NewProcess(sbmPid); err == nil {
		cpuPercent, _ := sbmProc.CPUPercent()
		var memoryMB float64
		if memInfo, err := sbmProc.MemoryInfo(); err == nil && memInfo != nil {
			memoryMB = float64(memInfo.RSS) / 1024 / 1024
		}

		result["sbm"] = ProcessStats{
			PID:        int(sbmPid),
			CPUPercent: cpuPercent,
			MemoryMB:   memoryMB,
		}
	}

	// 获取 sing-box 进程信息
	if s.processManager.IsRunning() {
		singboxPid := int32(s.processManager.GetPID())
		if singboxProc, err := process.NewProcess(singboxPid); err == nil {
			cpuPercent, _ := singboxProc.CPUPercent()
			var memoryMB float64
			if memInfo, err := singboxProc.MemoryInfo(); err == nil && memInfo != nil {
				memoryMB = float64(memInfo.RSS) / 1024 / 1024
			}

			result["singbox"] = ProcessStats{
				PID:        int(singboxPid),
				CPUPercent: cpuPercent,
				MemoryMB:   memoryMB,
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (s *Server) getDNSMonitor(c *gin.Context) {
	stats, logs, running := s.dnsService.Snapshot()
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"running": running, "stats": stats, "logs": logs}})
}

func (s *Server) getLogs(c *gin.Context) {
	lines := 200 // 默认返回 200 行
	if linesParam := c.Query("lines"); linesParam != "" {
		if n, err := strconv.Atoi(linesParam); err == nil && n > 0 {
			lines = n
		}
	}

	// 返回程序日志，不混合 sing-box 输出
	logs, err := logger.ReadAppLogs(lines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": logs})
}

// getAppLogs 获取应用日志
func (s *Server) getAppLogs(c *gin.Context) {
	lines := 200 // 默认返回 200 行
	if linesParam := c.Query("lines"); linesParam != "" {
		if n, err := strconv.Atoi(linesParam); err == nil && n > 0 {
			lines = n
		}
	}

	logs, err := logger.ReadAppLogs(lines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": logs})
}

// getSingboxLogs 获取 sing-box 日志
func (s *Server) getSingboxLogs(c *gin.Context) {
	lines := 200 // 默认返回 200 行
	if linesParam := c.Query("lines"); linesParam != "" {
		if n, err := strconv.Atoi(linesParam); err == nil && n > 0 {
			lines = n
		}
	}

	logs, err := logger.ReadSingboxLogs(lines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": logs})
}

// ==================== 节点 API ====================

func (s *Server) getAllNodes(c *gin.Context) {
	nodes := s.store.GetAllNodes()
	c.JSON(http.StatusOK, gin.H{"data": nodes})
}

func (s *Server) getCountryGroups(c *gin.Context) {
	groups := s.store.GetCountryGroups()
	c.JSON(http.StatusOK, gin.H{"data": groups})
}

func (s *Server) getNodesByCountry(c *gin.Context) {
	code := c.Param("code")
	nodes := s.store.GetNodesByCountry(code)
	c.JSON(http.StatusOK, gin.H{"data": nodes})
}

func (s *Server) parseNodeURL(c *gin.Context) {
	var req struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	node, err := parser.ParseURL(req.URL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "解析失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": node})
}

// ==================== 手动节点 API ====================

func (s *Server) getManualNodes(c *gin.Context) {
	nodes := s.store.GetManualNodes()
	c.JSON(http.StatusOK, gin.H{"data": nodes})
}

func (s *Server) addManualNode(c *gin.Context) {
	var node storage.ManualNode
	if err := c.ShouldBindJSON(&node); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 生成 ID
	node.ID = uuid.New().String()

	if err := s.store.AddManualNode(node); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 自动应用配置
	if err := s.autoApplyConfig(); err != nil {
		c.JSON(http.StatusOK, gin.H{"data": node, "warning": "添加成功，但自动应用配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": node})
}

func (s *Server) updateManualNode(c *gin.Context) {
	id := c.Param("id")

	var node storage.ManualNode
	if err := c.ShouldBindJSON(&node); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	node.ID = id
	if err := s.store.UpdateManualNode(node); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 自动应用配置
	if err := s.autoApplyConfig(); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "更新成功，但自动应用配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

func (s *Server) deleteManualNode(c *gin.Context) {
	id := c.Param("id")

	if err := s.store.DeleteManualNode(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 自动应用配置
	if err := s.autoApplyConfig(); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "删除成功，但自动应用配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ==================== 内核管理 API ====================

func (s *Server) getKernelInfo(c *gin.Context) {
	info := s.kernelManager.GetInfo()
	c.JSON(http.StatusOK, gin.H{"data": info})
}

func (s *Server) getKernelReleases(c *gin.Context) {
	releases, err := s.kernelManager.FetchReleases()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 只返回版本号和名称，不返回完整的 assets
	type ReleaseInfo struct {
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
	}

	result := make([]ReleaseInfo, len(releases))
	for i, r := range releases {
		result[i] = ReleaseInfo{
			TagName: r.TagName,
			Name:    r.Name,
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (s *Server) startKernelDownload(c *gin.Context) {
	var req struct {
		Version string `json:"version" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.kernelManager.StartDownload(req.Version); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "下载已开始"})
}

func (s *Server) getKernelProgress(c *gin.Context) {
	progress := s.kernelManager.GetProgress()
	c.JSON(http.StatusOK, gin.H{"data": progress})
}

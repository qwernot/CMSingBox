package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

const defaultAdminPassword = "admin"

var ErrSubscriptionLimitExceeded = errors.New("订阅链接数量已达到授权上限")

func defaultAuthConfig() (*AuthConfig, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(defaultAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("生成默认密码哈希失败: %w", err)
	}
	return &AuthConfig{Username: "admin", PasswordHash: string(hash)}, nil
}

// JSONStore JSON 文件存储实现
type JSONStore struct {
	dataDir           string
	mu                sync.RWMutex
	data              *AppData
	subscriptionLimit int
}

// NewJSONStore 创建新的 JSON 存储
func NewJSONStore(dataDir string) (*JSONStore, error) {
	store := &JSONStore{
		dataDir:           dataDir,
		subscriptionLimit: 1,
	}

	// 确保数据目录存在
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	// 确保 generated 子目录存在
	generatedDir := filepath.Join(dataDir, "generated")
	if err := os.MkdirAll(generatedDir, 0755); err != nil {
		return nil, fmt.Errorf("创建 generated 目录失败: %w", err)
	}

	// 加载数据
	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

// load 加载数据
func (s *JSONStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dataFile := filepath.Join(s.dataDir, "data.json")

	// 如果文件不存在，初始化默认数据
	if _, err := os.Stat(dataFile); os.IsNotExist(err) {
		auth, err := defaultAuthConfig()
		if err != nil {
			return err
		}
		s.data = &AppData{
			Subscriptions: []Subscription{},
			ManualNodes:   []ManualNode{},
			Filters:       []Filter{},
			Rules:         []Rule{},
			RuleGroups:    DefaultRuleGroups(),
			Settings:      DefaultSettings(),
			Auth:          auth,
		}
		return s.saveInternal()
	}

	// 读取文件
	data, err := os.ReadFile(dataFile)
	if err != nil {
		return fmt.Errorf("读取数据文件失败: %w", err)
	}

	s.data = &AppData{}
	if err := json.Unmarshal(data, s.data); err != nil {
		return fmt.Errorf("解析数据文件失败: %w", err)
	}

	// 确保 Settings 不为空
	if s.data.Settings == nil {
		s.data.Settings = DefaultSettings()
	}

	// 确保 RuleGroups 不为空
	if len(s.data.RuleGroups) == 0 {
		s.data.RuleGroups = DefaultRuleGroups()
	}

	// 迁移旧的路径格式（移除多余的 data/ 前缀）
	needSave := false
	// v1.0.5 之前没有独立的认证开关。已有代理凭据表示原配置启用了认证；
	// 仅在字段完全缺失时迁移，避免用户明确关闭后重启又被自动打开。
	if !bytes.Contains(data, []byte(`"mixed_auth_enabled"`)) {
		s.data.Settings.MixedAuthEnabled = s.data.Settings.MixedUsername != "" && s.data.Settings.MixedPassword != ""
		needSave = true
	}
	if s.data.Auth == nil || s.data.Auth.Username == "" || s.data.Auth.PasswordHash == "" {
		auth, err := defaultAuthConfig()
		if err != nil {
			return err
		}
		s.data.Auth = auth
		needSave = true
	}
	if s.data.Settings.DNSListen == "" {
		s.data.Settings.DNSListen = "0.0.0.0:53"
		s.data.Settings.DNSProxyUpstream = "127.0.0.1:1053"
		s.data.Settings.DNSDirectUpstream = "223.5.5.5:53"
		s.data.Settings.DNSRoutingMode = "default_proxy"
		s.data.Settings.DNSExceptions = []string{}
		needSave = true
	}
	if s.data.Settings.DNSListen == "0.0.0.0:5353" || s.data.Settings.DNSListen == ":5353" {
		s.data.Settings.DNSListen = "0.0.0.0:53"
		needSave = true
	}
	if s.data.Settings.FakeIPRange == "" {
		s.data.Settings.FakeIPRange = "198.18.0.0/15"
		s.data.Settings.LogEnabled = true
		s.data.Settings.LogLevel = "info"
		s.data.Settings.LogTimestamp = true
		s.data.Settings.ExtraInbounds = []map[string]interface{}{}
		s.data.Settings.ExtraOutbounds = []map[string]interface{}{}
		needSave = true
	}
	if s.data.Settings.TProxyPort == 0 {
		s.data.Settings.TProxyPort = 7893
		s.data.Settings.BypassCIDRs = []string{"0.0.0.0/8", "10.0.0.0/8", "127.0.0.0/8", "169.254.0.0/16", "172.16.0.0/12", "192.168.0.0/16", "224.0.0.0/4"}
		needSave = true
	}
	if s.data.Settings.ClientConfigPath == "" {
		s.data.Settings.ClientConfigPath = randomClientPath()
		needSave = true
	}
	if s.data.Settings.ClashAPISecret == "" {
		s.data.Settings.ClashAPISecret = randomClientPath()
		needSave = true
	}
	if s.data.Settings.BackHomePort == 0 {
		s.data.Settings.BackHomePort = 8443
		needSave = true
	}
	if s.data.Settings.SingBoxPath == "data/bin/sing-box" {
		s.data.Settings.SingBoxPath = "bin/sing-box"
		needSave = true
	}
	if s.data.Settings.ConfigPath == "data/generated/config.json" {
		s.data.Settings.ConfigPath = "generated/config.json"
		needSave = true
	}
	if needSave {
		return s.saveInternal()
	}

	return nil
}

// saveInternal 内部保存方法（不加锁）
func (s *JSONStore) saveInternal() error {
	dataFile := filepath.Join(s.dataDir, "data.json")

	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化数据失败: %w", err)
	}

	tempFile, err := os.CreateTemp(s.dataDir, "data-*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时数据文件失败: %w", err)
	}
	tempName := tempFile.Name()
	defer os.Remove(tempName)
	if err := tempFile.Chmod(0600); err != nil {
		tempFile.Close()
		return fmt.Errorf("设置数据文件权限失败: %w", err)
	}
	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return fmt.Errorf("写入临时数据文件失败: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		return fmt.Errorf("同步数据文件失败: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("关闭数据文件失败: %w", err)
	}
	if err := os.Rename(tempName, dataFile); err != nil {
		return fmt.Errorf("替换数据文件失败: %w", err)
	}

	return nil
}

// ExportConfiguration 导出不含认证密钥的配置快照。
func (s *JSONStore) ExportConfiguration() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	copyData := *s.data
	copyData.Auth = nil
	return json.MarshalIndent(&copyData, "", "  ")
}

// RestoreConfiguration 恢复配置快照。认证信息始终保留为当前值。
func (s *JSONStore) RestoreConfiguration(raw []byte) error {
	var restored AppData
	if err := json.Unmarshal(raw, &restored); err != nil {
		return fmt.Errorf("解析备份数据失败: %w", err)
	}
	if restored.Settings == nil {
		return fmt.Errorf("备份缺少 settings 配置")
	}
	if restored.Subscriptions == nil {
		restored.Subscriptions = []Subscription{}
	}
	if restored.ManualNodes == nil {
		restored.ManualNodes = []ManualNode{}
	}
	if restored.Filters == nil {
		restored.Filters = []Filter{}
	}
	if restored.Rules == nil {
		restored.Rules = []Rule{}
	}
	if len(restored.RuleGroups) == 0 {
		restored.RuleGroups = DefaultRuleGroups()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	restored.Auth = s.data.Auth
	if s.subscriptionLimit >= 0 && len(restored.Subscriptions) > s.subscriptionLimit {
		return fmt.Errorf("%w：当前允许 %d 条，备份包含 %d 条", ErrSubscriptionLimitExceeded, s.subscriptionLimit, len(restored.Subscriptions))
	}
	previous := s.data
	s.data = &restored
	if err := s.saveInternal(); err != nil {
		s.data = previous
		return err
	}
	return nil
}

// Save 保存数据
func (s *JSONStore) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveInternal()
}

// ==================== 订阅操作 ====================

// GetSubscriptions 获取所有订阅
func (s *JSONStore) GetSubscriptions() []Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Subscription, len(s.data.Subscriptions))
	for i := range s.data.Subscriptions {
		result[i] = cloneSubscription(s.data.Subscriptions[i])
	}
	return result
}

// GetSubscription 获取单个订阅
func (s *JSONStore) GetSubscription(id string) *Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.data.Subscriptions {
		if s.data.Subscriptions[i].ID == id {
			result := cloneSubscription(s.data.Subscriptions[i])
			return &result
		}
	}
	return nil
}

// AddSubscription 添加订阅
func (s *JSONStore) AddSubscription(sub Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.subscriptionLimit >= 0 && len(s.data.Subscriptions) >= s.subscriptionLimit {
		return fmt.Errorf("%w：当前最多允许 %d 条", ErrSubscriptionLimitExceeded, s.subscriptionLimit)
	}
	s.data.Subscriptions = append(s.data.Subscriptions, sub)
	return s.saveInternal()
}

// SetSubscriptionLimit 设置可保存的订阅链接上限。小于 0 表示不限制。
func (s *JSONStore) SetSubscriptionLimit(limit int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscriptionLimit = limit
}

func (s *JSONStore) SubscriptionUsage() (used, limit int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data.Subscriptions), s.subscriptionLimit
}

func (s *JSONStore) CanAddSubscription() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.subscriptionLimit >= 0 && len(s.data.Subscriptions) >= s.subscriptionLimit {
		return fmt.Errorf("%w：当前最多允许 %d 条", ErrSubscriptionLimitExceeded, s.subscriptionLimit)
	}
	return nil
}

func cloneSubscription(sub Subscription) Subscription {
	sub.Nodes = append([]Node(nil), sub.Nodes...)
	for i := range sub.Nodes {
		if sub.Nodes[i].Extra != nil {
			extra := make(map[string]interface{}, len(sub.Nodes[i].Extra))
			for key, value := range sub.Nodes[i].Extra {
				extra[key] = value
			}
			sub.Nodes[i].Extra = extra
		}
	}
	if sub.Traffic != nil {
		traffic := *sub.Traffic
		sub.Traffic = &traffic
	}
	if sub.ExpireAt != nil {
		expireAt := *sub.ExpireAt
		sub.ExpireAt = &expireAt
	}
	return sub
}

// UpdateSubscription 更新订阅
func (s *JSONStore) UpdateSubscription(sub Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.data.Subscriptions {
		if s.data.Subscriptions[i].ID == sub.ID {
			s.data.Subscriptions[i] = sub
			return s.saveInternal()
		}
	}
	return fmt.Errorf("订阅不存在: %s", sub.ID)
}

// DeleteSubscription 删除订阅
func (s *JSONStore) DeleteSubscription(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.data.Subscriptions {
		if s.data.Subscriptions[i].ID == id {
			s.data.Subscriptions = append(s.data.Subscriptions[:i], s.data.Subscriptions[i+1:]...)
			return s.saveInternal()
		}
	}
	return fmt.Errorf("订阅不存在: %s", id)
}

// ==================== 过滤器操作 ====================

// GetFilters 获取所有过滤器
func (s *JSONStore) GetFilters() []Filter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Filters
}

// GetFilter 获取单个过滤器
func (s *JSONStore) GetFilter(id string) *Filter {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.data.Filters {
		if s.data.Filters[i].ID == id {
			return &s.data.Filters[i]
		}
	}
	return nil
}

// AddFilter 添加过滤器
func (s *JSONStore) AddFilter(filter Filter) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.Filters = append(s.data.Filters, filter)
	return s.saveInternal()
}

// UpdateFilter 更新过滤器
func (s *JSONStore) UpdateFilter(filter Filter) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.data.Filters {
		if s.data.Filters[i].ID == filter.ID {
			s.data.Filters[i] = filter
			return s.saveInternal()
		}
	}
	return fmt.Errorf("过滤器不存在: %s", filter.ID)
}

// DeleteFilter 删除过滤器
func (s *JSONStore) DeleteFilter(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.data.Filters {
		if s.data.Filters[i].ID == id {
			s.data.Filters = append(s.data.Filters[:i], s.data.Filters[i+1:]...)
			return s.saveInternal()
		}
	}
	return fmt.Errorf("过滤器不存在: %s", id)
}

// ==================== 规则操作 ====================

// GetRules 获取所有自定义规则
func (s *JSONStore) GetRules() []Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Rules
}

// AddRule 添加规则
func (s *JSONStore) AddRule(rule Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.Rules = append(s.data.Rules, rule)
	return s.saveInternal()
}

// UpdateRule 更新规则
func (s *JSONStore) UpdateRule(rule Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.data.Rules {
		if s.data.Rules[i].ID == rule.ID {
			s.data.Rules[i] = rule
			return s.saveInternal()
		}
	}
	return fmt.Errorf("规则不存在: %s", rule.ID)
}

// DeleteRule 删除规则
func (s *JSONStore) DeleteRule(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.data.Rules {
		if s.data.Rules[i].ID == id {
			s.data.Rules = append(s.data.Rules[:i], s.data.Rules[i+1:]...)
			return s.saveInternal()
		}
	}
	return fmt.Errorf("规则不存在: %s", id)
}

// ==================== 规则组操作 ====================

// GetRuleGroups 获取所有预设规则组
func (s *JSONStore) GetRuleGroups() []RuleGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.RuleGroups
}

// UpdateRuleGroup 更新规则组
func (s *JSONStore) UpdateRuleGroup(ruleGroup RuleGroup) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.data.RuleGroups {
		if s.data.RuleGroups[i].ID == ruleGroup.ID {
			s.data.RuleGroups[i] = ruleGroup
			return s.saveInternal()
		}
	}
	return fmt.Errorf("规则组不存在: %s", ruleGroup.ID)
}

// ==================== 设置操作 ====================

// GetSettings 获取设置
func (s *JSONStore) GetSettings() *Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Settings
}

// UpdateSettings 更新设置
func (s *JSONStore) UpdateSettings(settings *Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.Settings = settings
	return s.saveInternal()
}

// GetAuthConfig 获取认证配置的副本，避免调用方修改存储中的数据。
func (s *JSONStore) GetAuthConfig() AuthConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return *s.data.Auth
}

// UpdateAuthConfig 更新认证配置。
func (s *JSONStore) UpdateAuthConfig(auth AuthConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Auth = &auth
	return s.saveInternal()
}

// ==================== 手动节点操作 ====================

// GetManualNodes 获取所有手动节点
func (s *JSONStore) GetManualNodes() []ManualNode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.ManualNodes
}

// AddManualNode 添加手动节点
func (s *JSONStore) AddManualNode(node ManualNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.ManualNodes = append(s.data.ManualNodes, node)
	return s.saveInternal()
}

// UpdateManualNode 更新手动节点
func (s *JSONStore) UpdateManualNode(node ManualNode) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.data.ManualNodes {
		if s.data.ManualNodes[i].ID == node.ID {
			s.data.ManualNodes[i] = node
			return s.saveInternal()
		}
	}
	return fmt.Errorf("手动节点不存在: %s", node.ID)
}

// DeleteManualNode 删除手动节点
func (s *JSONStore) DeleteManualNode(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.data.ManualNodes {
		if s.data.ManualNodes[i].ID == id {
			s.data.ManualNodes = append(s.data.ManualNodes[:i], s.data.ManualNodes[i+1:]...)
			return s.saveInternal()
		}
	}
	return fmt.Errorf("手动节点不存在: %s", id)
}

// ==================== 辅助方法 ====================

// GetAllNodes 获取所有启用的节点（订阅节点 + 手动节点）
func (s *JSONStore) GetAllNodes() []Node {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var nodes []Node
	// 添加订阅节点
	for _, sub := range s.data.Subscriptions {
		if sub.Enabled {
			nodes = append(nodes, sub.Nodes...)
		}
	}
	// 添加手动节点
	for _, mn := range s.data.ManualNodes {
		if mn.Enabled {
			nodes = append(nodes, mn.Node)
		}
	}
	return nodes
}

// GetNodesByCountry 按国家获取节点
func (s *JSONStore) GetNodesByCountry(countryCode string) []Node {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var nodes []Node
	// 订阅节点
	for _, sub := range s.data.Subscriptions {
		if sub.Enabled {
			for _, node := range sub.Nodes {
				if node.Country == countryCode {
					nodes = append(nodes, node)
				}
			}
		}
	}
	// 手动节点
	for _, mn := range s.data.ManualNodes {
		if mn.Enabled && mn.Node.Country == countryCode {
			nodes = append(nodes, mn.Node)
		}
	}
	return nodes
}

// GetCountryGroups 获取所有国家节点分组
func (s *JSONStore) GetCountryGroups() []CountryGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()

	countryCount := make(map[string]int)

	// 统计订阅节点
	for _, sub := range s.data.Subscriptions {
		if sub.Enabled {
			for _, node := range sub.Nodes {
				if node.Country != "" {
					countryCount[node.Country]++
				}
			}
		}
	}
	// 统计手动节点
	for _, mn := range s.data.ManualNodes {
		if mn.Enabled && mn.Node.Country != "" {
			countryCount[mn.Node.Country]++
		}
	}

	var groups []CountryGroup
	for code, count := range countryCount {
		groups = append(groups, CountryGroup{
			Code:      code,
			Name:      GetCountryName(code),
			Emoji:     GetCountryEmoji(code),
			NodeCount: count,
		})
	}

	return groups
}

// GetDataDir 获取数据目录
func (s *JSONStore) GetDataDir() string {
	return s.dataDir
}

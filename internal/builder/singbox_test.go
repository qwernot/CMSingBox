package builder

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"

	"cmsingbox.local/cmsingbox/internal/storage"
)

func TestConfigBuilder_NodeToOutbound_TUICEnsuresTLS(t *testing.T) {
	b := NewConfigBuilder(storage.DefaultSettings(), nil, nil, nil, nil)

	outbound := b.nodeToOutbound(storage.Node{
		Tag:        "tuic-node",
		Type:       "tuic",
		Server:     "example.com",
		ServerPort: 443,
		Extra: map[string]interface{}{
			"uuid":     "11111111-1111-1111-1111-111111111111",
			"password": "secret",
		},
	})

	tls, ok := outbound["tls"].(map[string]interface{})
	if !ok {
		t.Fatalf("outbound[\"tls\"] type = %T, want map[string]interface{}", outbound["tls"])
	}
	if enabled, ok := tls["enabled"].(bool); !ok || !enabled {
		t.Fatalf("tls.enabled = %#v, want true", tls["enabled"])
	}
}

func TestConfigBuilderSkipsUnsupportedKCPNode(t *testing.T) {
	nodes := []storage.Node{{Tag: "legacy-kcp", Type: "vless", Server: "example.com", ServerPort: 443, Extra: map[string]interface{}{"uuid": "11111111-1111-1111-1111-111111111111", "transport": map[string]interface{}{"type": "kcp"}}}}
	config, err := NewConfigBuilder(storage.DefaultSettings(), nodes, nil, nil, nil).BuildJSON()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(config, "legacy-kcp") {
		t.Fatal("unsupported KCP node should be omitted")
	}
}

func TestConfigBuilder_BuildJSON_OmitsLegacyInboundFieldsByDefault(t *testing.T) {
	settings := storage.DefaultSettings()
	settings.DNSEnabled = false
	settings.TunEnabled = true

	b := NewConfigBuilder(settings, nil, nil, nil, nil)

	configJSON, err := b.BuildJSON()
	if err != nil {
		t.Fatalf("BuildJSON() error = %v", err)
	}

	var config map[string]any
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	inbounds, ok := config["inbounds"].([]any)
	if !ok {
		t.Fatalf("config[\"inbounds\"] type = %T, want []any", config["inbounds"])
	}
	if len(inbounds) != 2 {
		t.Fatalf("len(inbounds) = %d, want 2", len(inbounds))
	}

	for i, inboundRaw := range inbounds {
		inbound, ok := inboundRaw.(map[string]any)
		if !ok {
			t.Fatalf("inbounds[%d] type = %T, want map[string]any", i, inboundRaw)
		}
		if _, exists := inbound["sniff"]; exists {
			t.Fatalf("inbounds[%d] unexpectedly contains legacy field \"sniff\": %#v", i, inbound["sniff"])
		}
		if _, exists := inbound["sniff_override_destination"]; exists {
			t.Fatalf("inbounds[%d] unexpectedly contains legacy field \"sniff_override_destination\": %#v", i, inbound["sniff_override_destination"])
		}
	}
}

func TestConfigBuilder_LinuxTunKeepsHostServicesOutsideProxy(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only TUN routing fields")
	}
	settings := storage.DefaultSettings()
	settings.DNSEnabled = false
	settings.TunEnabled = true
	config, err := NewConfigBuilder(settings, nil, nil, nil, nil).Build()
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Inbounds) < 2 {
		t.Fatal("tun inbound missing")
	}
	tun, ok := config.Inbounds[1].(Inbound)
	if !ok {
		t.Fatalf("tun inbound type = %T", config.Inbounds[1])
	}
	if !tun.AutoRedirect || len(tun.ExcludeUID) != 1 || tun.ExcludeUID[0] != 0 {
		t.Fatalf("unexpected Linux TUN host bypass: %#v", tun)
	}
}

func TestConfigBuilder_WithSingBoxVersion_KeepsLegacyInboundFieldsForPre113(t *testing.T) {
	settings := storage.DefaultSettings()
	settings.DNSEnabled = false
	settings.TunEnabled = true

	b := NewConfigBuilder(settings, nil, nil, nil, nil).
		WithSingBoxVersion("sing-box version 1.12.12")

	configJSON, err := b.BuildJSON()
	if err != nil {
		t.Fatalf("BuildJSON() error = %v", err)
	}

	var config map[string]any
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	inbounds, ok := config["inbounds"].([]any)
	if !ok {
		t.Fatalf("config[\"inbounds\"] type = %T, want []any", config["inbounds"])
	}

	for i, inboundRaw := range inbounds {
		inbound, ok := inboundRaw.(map[string]any)
		if !ok {
			t.Fatalf("inbounds[%d] type = %T, want map[string]any", i, inboundRaw)
		}
		if value, exists := inbound["sniff"]; !exists || value != true {
			t.Fatalf("inbounds[%d].sniff = %#v, want true", i, value)
		}
		if value, exists := inbound["sniff_override_destination"]; !exists || value != true {
			t.Fatalf("inbounds[%d].sniff_override_destination = %#v, want true", i, value)
		}
	}
}

func TestConfigBuilder_WithSingBoxVersion_RemovesLegacyTProxyFieldsFor113OrLater(t *testing.T) {
	settings := storage.DefaultSettings()
	settings.TransparentProxy = true

	configJSON, err := NewConfigBuilder(settings, nil, nil, nil, nil).
		WithSingBoxVersion("sing-box version 1.13.21").
		BuildJSON()
	if err != nil {
		t.Fatal(err)
	}

	var config map[string]any
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		t.Fatal(err)
	}
	inbounds := config["inbounds"].([]any)
	tproxy := inbounds[len(inbounds)-1].(map[string]any)
	if tproxy["type"] != "tproxy" {
		t.Fatalf("last inbound type = %#v, want tproxy", tproxy["type"])
	}
	if _, exists := tproxy["sniff"]; exists {
		t.Fatalf("tproxy inbound unexpectedly contains legacy sniff field: %#v", tproxy)
	}
	if _, exists := tproxy["sniff_override_destination"]; exists {
		t.Fatalf("tproxy inbound unexpectedly contains legacy sniff_override_destination field: %#v", tproxy)
	}
}

func TestCompatProfileFromVersion_UsesModernProfileFor113OrLater(t *testing.T) {
	profile := CompatProfileFromVersion("sing-box version 1.13.5")
	if profile.LegacyInboundFields {
		t.Fatalf("LegacyInboundFields = %v, want false", profile.LegacyInboundFields)
	}
}

func TestConfigBuilder_DomainDNSServerUsesBootstrapResolver(t *testing.T) {
	settings := storage.DefaultSettings()
	settings.DirectDNS = "https://dns.alidns.com/dns-query"
	configJSON, err := NewConfigBuilder(settings, nil, nil, nil, nil).BuildJSON()
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		DNS struct {
			Servers []DNSServer `json:"servers"`
		} `json:"dns"`
	}
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		t.Fatal(err)
	}
	var direct *DNSServer
	for i := range config.DNS.Servers {
		if config.DNS.Servers[i].Tag == "dns_direct" {
			direct = &config.DNS.Servers[i]
		}
	}
	if direct == nil || direct.DomainResolver == nil || direct.DomainResolver.Server != "dns_bootstrap" {
		t.Fatalf("dns_direct domain resolver = %#v, want dns_bootstrap", direct)
	}
}

func TestConfigBuilder_DefaultDomainResolverUsesBootstrap(t *testing.T) {
	settings := storage.DefaultSettings()
	config, err := NewConfigBuilder(settings, nil, nil, nil, nil).Build()
	if err != nil {
		t.Fatal(err)
	}
	if config.Route == nil || config.Route.DefaultDomainResolver == nil {
		t.Fatal("route.default_domain_resolver is missing")
	}
	if got := config.Route.DefaultDomainResolver.Server; got != "dns_bootstrap" {
		t.Fatalf("route.default_domain_resolver.server = %q, want dns_bootstrap", got)
	}
	var bootstrap *DNSServer
	for i := range config.DNS.Servers {
		if config.DNS.Servers[i].Tag == "dns_bootstrap" {
			bootstrap = &config.DNS.Servers[i]
			break
		}
	}
	if bootstrap == nil || bootstrap.Type != "udp" || bootstrap.Server != "223.5.5.5" || bootstrap.ServerPort != 53 {
		t.Fatalf("dns_bootstrap = %#v, want configured direct DNS", bootstrap)
	}
}

func TestConfigBuilder_AppliesDNSStrategyAndClashUISettings(t *testing.T) {
	settings := storage.DefaultSettings()
	settings.DNSStrategy = "ipv4_only"
	settings.ClashUIPath = "zashboard"
	settings.ClashUIURL = "https://example.com/zashboard.zip"
	settings.ClashUIDetour = "Proxy"

	config, err := NewConfigBuilder(settings, nil, nil, nil, nil).Build()
	if err != nil {
		t.Fatal(err)
	}
	if config.DNS == nil || config.DNS.Strategy != "ipv4_only" {
		t.Fatalf("dns strategy = %#v, want ipv4_only", config.DNS)
	}
	api := config.Experimental.ClashAPI
	if api.ExternalUI != "custom-ui-0" || api.ExternalUIDownloadURL != settings.ClashUIURL || api.ExternalUIDownloadDetour != "Proxy" {
		t.Fatalf("unexpected Clash UI config: %#v", api)
	}
}

func TestConfigBuilder_InvalidDNSStrategyFallsBackToPreferIPv4(t *testing.T) {
	settings := storage.DefaultSettings()
	settings.DNSStrategy = "invalid"

	config, err := NewConfigBuilder(settings, nil, nil, nil, nil).Build()
	if err != nil {
		t.Fatal(err)
	}
	if config.DNS == nil || config.DNS.Strategy != "prefer_ipv4" {
		t.Fatalf("dns strategy = %#v, want prefer_ipv4", config.DNS)
	}
}

func TestConfigBuilder_MixedInboundAuthentication(t *testing.T) {
	settings := storage.DefaultSettings()
	settings.AllowLAN = true
	settings.MixedAuthEnabled = true
	settings.MixedUsername = "proxy-user"
	settings.MixedPassword = "proxy-pass"
	config, err := NewConfigBuilder(settings, nil, nil, nil, nil).Build()
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Inbounds) == 0 {
		t.Fatal("mixed inbound missing")
	}
	mixed, ok := config.Inbounds[0].(Inbound)
	if !ok || mixed.Listen != "0.0.0.0" || len(mixed.Users) != 1 || mixed.Users[0].Username != "proxy-user" || mixed.Users[0].Password != "proxy-pass" {
		t.Fatalf("unexpected authenticated mixed inbound: %#v", config.Inbounds[0])
	}
}

func TestConfigBuilder_MixedInboundWithoutAuthentication(t *testing.T) {
	settings := storage.DefaultSettings()
	settings.AllowLAN = true
	settings.MixedAuthEnabled = false
	settings.MixedUsername = "saved-user"
	settings.MixedPassword = "saved-pass"
	config, err := NewConfigBuilder(settings, nil, nil, nil, nil).Build()
	if err != nil {
		t.Fatal(err)
	}
	mixed, ok := config.Inbounds[0].(Inbound)
	if !ok || mixed.Listen != "0.0.0.0" || len(mixed.Users) != 0 {
		t.Fatalf("unexpected unauthenticated mixed inbound: %#v", config.Inbounds[0])
	}
}

func TestConfigBuilder_EmbeddedProxyDNSInbound(t *testing.T) {
	settings := storage.DefaultSettings()
	settings.DNSEnabled = true
	settings.DNSProxyUpstream = "127.0.0.1:1053"

	config, err := NewConfigBuilder(settings, nil, nil, nil, nil).Build()
	if err != nil {
		t.Fatal(err)
	}

	foundInbound := false
	for _, raw := range config.Inbounds {
		inbound, ok := raw.(Inbound)
		if !ok || inbound.Tag != embeddedDNSInboundTag {
			continue
		}
		foundInbound = true
		if inbound.Type != "direct" || inbound.Listen != "127.0.0.1" || inbound.ListenPort != 1053 {
			t.Fatalf("unexpected embedded DNS inbound: %#v", inbound)
		}
	}
	if !foundInbound {
		t.Fatal("embedded DNS inbound missing")
	}

	foundRule := false
	for _, rule := range config.Route.Rules {
		if rule["inbound"] == embeddedDNSInboundTag && rule["action"] == "hijack-dns" {
			foundRule = true
			break
		}
	}
	if !foundRule {
		t.Fatal("embedded DNS hijack rule missing")
	}
}

func TestConfigBuilder_ExternalProxyDNSDoesNotBindEmbeddedPort(t *testing.T) {
	settings := storage.DefaultSettings()
	settings.DNSEnabled = true
	settings.DNSProxyUpstream = "192.168.1.2:5353"

	config, err := NewConfigBuilder(settings, nil, nil, nil, nil).Build()
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range config.Inbounds {
		if inbound, ok := raw.(Inbound); ok && inbound.Tag == embeddedDNSInboundTag {
			t.Fatalf("unexpected embedded DNS inbound for external upstream: %#v", inbound)
		}
	}
}

func TestConfigBuilder_ClashAPIIsReachableAndProtected(t *testing.T) {
	settings := storage.DefaultSettings()
	settings.AllowLAN = false
	settings.ClashAPIPort = 19091
	settings.ClashAPISecret = "console-secret"

	config, err := NewConfigBuilder(settings, nil, nil, nil, nil).Build()
	if err != nil {
		t.Fatal(err)
	}
	if config.Experimental == nil || config.Experimental.ClashAPI == nil {
		t.Fatal("Clash API configuration missing")
	}
	api := config.Experimental.ClashAPI
	if api.ExternalController != "0.0.0.0:19091" {
		t.Fatalf("external_controller = %q, want %q", api.ExternalController, "0.0.0.0:19091")
	}
	if api.Secret != "console-secret" {
		t.Fatalf("secret = %q, want configured secret", api.Secret)
	}
	if api.ExternalUIDownloadURL != "" {
		t.Fatalf("external UI should be bundled, download URL = %q", api.ExternalUIDownloadURL)
	}
	if api.DefaultMode != "rule" {
		t.Fatalf("default_mode = %q, want rule", api.DefaultMode)
	}
}

func TestConfigBuilder_DefaultRulesUseProjectMirror(t *testing.T) {
	config, err := NewConfigBuilder(storage.DefaultSettings(), nil, nil, nil, storage.DefaultRuleGroups()).Build()
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Route.RuleSet) == 0 {
		t.Fatal("default rule sets missing")
	}
	for _, ruleSet := range config.Route.RuleSet {
		if !strings.Contains(ruleSet.URL, "raw.githubusercontent.com/qwernot/CM/main/rules/") {
			t.Fatalf("rule set still depends on a third-party mirror: %s", ruleSet.URL)
		}
	}
}

func TestConfigBuilder_DuplicateNodeTagsBecomeUnique(t *testing.T) {
	nodes := []storage.Node{
		{Tag: "日本节点", Type: "socks", Server: "127.0.0.1", ServerPort: 10001},
		{Tag: "日本节点", Type: "socks", Server: "127.0.0.1", ServerPort: 10002},
		{Tag: "Proxy", Type: "socks", Server: "127.0.0.1", ServerPort: 10003},
	}
	config, err := NewConfigBuilder(storage.DefaultSettings(), nodes, nil, nil, nil).Build()
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, outbound := range config.Outbounds {
		tag, _ := outbound["tag"].(string)
		if seen[tag] {
			t.Fatalf("duplicate outbound tag %q", tag)
		}
		seen[tag] = true
	}
	if !seen["日本节点"] || !seen["日本节点 (2)"] || !seen["Proxy (2)"] {
		t.Fatalf("normalized node tags missing: %#v", seen)
	}
}

func TestConfigBuilder_WithoutNodesUsesDirectFallback(t *testing.T) {
	config, err := NewConfigBuilder(storage.DefaultSettings(), nil, nil, nil, storage.DefaultRuleGroups()).Build()
	if err != nil {
		t.Fatal(err)
	}
	for _, outbound := range config.Outbounds {
		if outbound["tag"] != "Proxy" {
			continue
		}
		if outbound["default"] != "DIRECT" {
			t.Fatalf("Proxy default = %#v, want DIRECT", outbound["default"])
		}
		values, ok := outbound["outbounds"].([]string)
		if !ok || len(values) != 1 || values[0] != "DIRECT" {
			t.Fatalf("Proxy outbounds = %#v, want [DIRECT]", outbound["outbounds"])
		}
		return
	}
	t.Fatal("Proxy selector missing")
}

func TestConfigBuilder_UpdaterDomainsBypassFakeIP(t *testing.T) {
	config, err := NewConfigBuilder(storage.DefaultSettings(), nil, nil, nil, nil).Build()
	if err != nil {
		t.Fatal(err)
	}
	dnsFound := false
	for _, rule := range config.DNS.Rules {
		if len(rule.Domain) > 0 && rule.Domain[0] == "api.github.com" && rule.Server == "dns_direct" {
			dnsFound = true
			break
		}
	}
	if !dnsFound {
		t.Fatalf("updater DNS rule missing: %#v", config.DNS.Rules)
	}
	found := false
	for _, rule := range config.Route.Rules {
		domains, ok := rule["domain"].([]string)
		if ok && len(domains) > 0 && domains[0] == "api.github.com" && rule["outbound"] == "DIRECT" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("DIRECT updater route missing")
	}
}

func TestConfigBuilder_IP111UsesDirectRoute(t *testing.T) {
	config, err := NewConfigBuilder(storage.DefaultSettings(), nil, nil, nil, nil).Build()
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range config.Route.Rules {
		domains, ok := rule["domain_suffix"].([]string)
		if ok && len(domains) == 1 && domains[0] == "ip111.cn" && rule["outbound"] == "DIRECT" {
			return
		}
	}
	t.Fatal("ip111.cn DIRECT rule missing")
}

func TestConfigBuilder_CNDomainsDirectFallbackAfterUserRules(t *testing.T) {
	rules := []storage.Rule{{Name: "用户 .cn 规则", RuleType: "domain_suffix", Values: []string{"cn"}, Outbound: "Proxy", Enabled: true}}
	config, err := NewConfigBuilder(storage.DefaultSettings(), nil, nil, rules, nil).Build()
	if err != nil {
		t.Fatal(err)
	}
	userIndex, fallbackIndex := -1, -1
	for i, rule := range config.Route.Rules {
		suffixes, ok := rule["domain_suffix"].([]string)
		if !ok || len(suffixes) != 1 || suffixes[0] != "cn" {
			continue
		}
		if rule["outbound"] == "Proxy" {
			userIndex = i
		}
		if rule["outbound"] == "DIRECT" {
			fallbackIndex = i
		}
	}
	if userIndex < 0 || fallbackIndex <= userIndex {
		t.Fatalf(".cn fallback must follow user rules: user=%d fallback=%d", userIndex, fallbackIndex)
	}
}

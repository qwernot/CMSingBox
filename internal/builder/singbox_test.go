package builder

import (
	"encoding/json"
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

func TestConfigBuilder_BuildJSON_OmitsLegacyInboundFieldsByDefault(t *testing.T) {
	settings := storage.DefaultSettings()
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

func TestConfigBuilder_WithSingBoxVersion_KeepsLegacyInboundFieldsForPre113(t *testing.T) {
	settings := storage.DefaultSettings()
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

package storage

import "testing"

func TestGatewayDefaults(t *testing.T) {
	settings := DefaultSettings()
	if !settings.DNSEnabled || settings.DNSListen != "0.0.0.0:53" {
		t.Fatalf("unexpected DNS defaults: %+v", settings)
	}
	if settings.ProxyDNS != "https://8.8.8.8/dns-query" || settings.DirectDNS != "udp://223.5.5.5:53" || settings.DNSProxyUpstream != "127.0.0.1:1053" || settings.DNSDirectUpstream != "223.5.5.5:53" {
		t.Fatalf("unexpected resolver defaults: proxy=%s direct=%s internal=%s direct upstream=%s", settings.ProxyDNS, settings.DirectDNS, settings.DNSProxyUpstream, settings.DNSDirectUpstream)
	}
	if settings.ClashAPISecret != "" {
		t.Fatal("default Clash API must be passwordless")
	}
}

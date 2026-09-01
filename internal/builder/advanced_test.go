package builder

import (
	"strings"
	"testing"

	"cmsingbox.local/cmsingbox/internal/storage"
)

func TestAdvancedConfigurationIsGenerated(t *testing.T) {
	settings := storage.DefaultSettings()
	settings.ProxyDNS = "https://1.1.1.1/dns-query"
	settings.DirectDNS = "udp://223.5.5.5:53"
	settings.FakeIPRange = "198.19.0.0/16"
	settings.TransparentProxy = true
	settings.TProxyPort = 7893
	settings.ExtraInbounds = []map[string]interface{}{{"type": "http", "tag": "extra-http", "listen_port": 8080}}
	settings.ExtraOutbounds = []map[string]interface{}{{"type": "direct", "tag": "wan2", "bind_interface": "eth1"}}
	settings.BackHomeEnabled = true
	settings.BackHomeServer = "home.example.com"
	settings.BackHomePort = 8443
	settings.BackHomePassword = "secret"
	settings.BackHomeCertPath = "/cert.pem"
	settings.BackHomeKeyPath = "/key.pem"
	generated, err := NewConfigBuilder(settings, nil, nil, nil, nil).BuildJSON()
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"dns-query", "198.19.0.0/16", "tproxy-in", "extra-http", "wan2", "backhome-in", "home.example.com"} {
		if !strings.Contains(generated, expected) {
			t.Fatalf("generated config missing %q: %s", expected, generated)
		}
	}
}

package api

import (
	"strings"
	"testing"

	"cmsingbox.local/cmsingbox/internal/storage"
)

func TestValidateManualNodeRequiresProtocolFields(t *testing.T) {
	node := storage.ManualNode{Node: storage.Node{Tag: "test", Type: "shadowsocks", Server: "example.com", ServerPort: 443}}
	if err := validateManualNode(&node); err == nil || !strings.Contains(err.Error(), "method") {
		t.Fatalf("expected missing method, got %v", err)
	}
	node.Node.Extra = map[string]interface{}{"method": "aes-128-gcm", "password": "secret"}
	if err := validateManualNode(&node); err != nil {
		t.Fatalf("valid node rejected: %v", err)
	}
}

func TestValidateManualNodeAddsTLSForHysteria2(t *testing.T) {
	node := storage.ManualNode{Node: storage.Node{Tag: "home", Type: "hysteria2", Server: "example.com", ServerPort: 443, Extra: map[string]interface{}{"password": "secret"}}}
	if err := validateManualNode(&node); err != nil {
		t.Fatal(err)
	}
	if node.Node.Extra["tls"] == nil {
		t.Fatal("missing default TLS")
	}
}

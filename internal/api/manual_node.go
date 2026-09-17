package api

import (
	"fmt"
	"strings"

	"cmsingbox.local/cmsingbox/internal/storage"
)

// validateManualNode prevents storing a node that sing-box cannot possibly use.
func validateManualNode(manual *storage.ManualNode) error {
	node := &manual.Node
	node.Tag = strings.TrimSpace(node.Tag)
	node.Server = strings.TrimSpace(node.Server)
	if node.Tag == "" || node.Server == "" || node.ServerPort < 1 || node.ServerPort > 65535 {
		return fmt.Errorf("请填写节点名称、服务器和有效端口")
	}
	if node.Extra == nil {
		node.Extra = map[string]interface{}{}
	}
	require := func(key string) error {
		value, ok := node.Extra[key].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s 节点缺少 %s", node.Type, key)
		}
		return nil
	}
	switch node.Type {
	case "shadowsocks":
		if err := require("method"); err != nil {
			return err
		}
		return require("password")
	case "vmess", "vless":
		if err := require("uuid"); err != nil {
			return err
		}
		if node.Type == "vmess" {
			if _, ok := node.Extra["security"]; !ok {
				node.Extra["security"] = "auto"
			}
		}
	case "trojan", "hysteria2":
		if err := require("password"); err != nil {
			return err
		}
	case "tuic":
		if err := require("uuid"); err != nil {
			return err
		}
		if err := require("password"); err != nil {
			return err
		}
	case "socks":
		return nil
	default:
		return fmt.Errorf("不支持的节点类型: %s", node.Type)
	}
	if node.Type == "trojan" || node.Type == "hysteria2" || node.Type == "tuic" {
		if _, ok := node.Extra["tls"]; !ok {
			node.Extra["tls"] = map[string]interface{}{"enabled": true, "server_name": node.Server}
		}
	}
	return nil
}

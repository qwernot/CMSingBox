package parser

import (
	"fmt"
	"net/url"
	"strings"

	"cmsingbox.local/cmsingbox/internal/storage"
)

// AnyTLSParser 解析 anytls://password@server:port?... 分享链接。
type AnyTLSParser struct{}

func (p *AnyTLSParser) Protocol() string { return "anytls" }

func (p *AnyTLSParser) Parse(rawURL string) (*storage.Node, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("解析 AnyTLS URL 失败: %w", err)
	}
	if u.User == nil || u.User.Username() == "" {
		return nil, fmt.Errorf("AnyTLS 密码为空")
	}
	server, serverPort, err := parseServerInfo(u.Host)
	if err != nil {
		return nil, err
	}
	name, _ := url.QueryUnescape(u.Fragment)
	if name == "" {
		name = fmt.Sprintf("%s:%d", server, serverPort)
	}
	query := u.Query()
	serverName := query.Get("sni")
	if serverName == "" {
		serverName = server
	}
	tls := map[string]interface{}{
		"enabled":     true,
		"server_name": serverName,
	}
	if getParamBool(query, "insecure") || getParamBool(query, "allowInsecure") {
		tls["insecure"] = true
	}
	return &storage.Node{
		Tag:        name,
		Type:       "anytls",
		Server:     server,
		ServerPort: serverPort,
		Extra: map[string]interface{}{
			"password": u.User.Username(),
			"tls":      tls,
		},
	}, nil
}

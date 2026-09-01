package firewall

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const tableName = "singbox_manager"

type Config struct {
	Port        int
	BypassCIDRs []string
}

func BuildNFTables(config Config) (string, error) {
	if config.Port < 1 || config.Port > 65535 {
		return "", fmt.Errorf("无效 TProxy 端口")
	}
	var cidrs []string
	for _, value := range config.BypassCIDRs {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, _, err := net.ParseCIDR(value); err != nil {
			return "", fmt.Errorf("无效绕过网段 %s", value)
		}
		cidrs = append(cidrs, value)
	}
	if len(cidrs) == 0 {
		cidrs = []string{"127.0.0.0/8"}
	}
	return fmt.Sprintf(`table inet %s {
  set bypass_v4 { type ipv4_addr; flags interval; elements = { %s } }
  chain prerouting {
    type filter hook prerouting priority mangle; policy accept;
    meta mark 0xff return
    ip daddr @bypass_v4 return
    meta l4proto { tcp, udp } tproxy to :%d meta mark set 0x1 accept
  }
}
`, tableName, strings.Join(cidrs, ", "), config.Port), nil
}

type Manager struct{}

func New() *Manager { return &Manager{} }

func command(name string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (m *Manager) Supported() bool { _, err := exec.LookPath("nft"); return err == nil }
func (m *Manager) Active() bool    { return command("nft", "list", "table", "inet", tableName) == nil }
func (m *Manager) Apply(config Config) error {
	script, err := BuildNFTables(config)
	if err != nil {
		return err
	}
	if !m.Supported() {
		return fmt.Errorf("系统未安装 nftables")
	}
	_ = command("nft", "delete", "table", "inet", tableName)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "nft", "-f", "-")
	cmd.Stdin = bytes.NewBufferString(script)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("应用 nftables 失败: %s", strings.TrimSpace(string(output)))
	}
	_ = command("ip", "rule", "del", "fwmark", "1", "table", "100")
	if err := command("ip", "rule", "add", "fwmark", "1", "table", "100"); err != nil {
		return err
	}
	if err := command("ip", "route", "replace", "local", "default", "dev", "lo", "table", "100"); err != nil {
		return err
	}
	return nil
}
func (m *Manager) Disable() error {
	_ = command("ip", "rule", "del", "fwmark", "1", "table", "100")
	if !m.Supported() {
		return nil
	}
	return command("nft", "delete", "table", "inet", tableName)
}
func PortString(port int) string { return strconv.Itoa(port) }

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
const redirectChain = "CMSINGBOX_REDIRECT"

type Config struct {
	Port        int
	BypassCIDRs []string
	Backend     string
}

func validate(config Config) ([]string, error) {
	if config.Port < 1 || config.Port > 65535 {
		return nil, fmt.Errorf("无效透明代理端口")
	}
	var cidrs []string
	for _, value := range config.BypassCIDRs {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		ip, _, err := net.ParseCIDR(value)
		if err != nil {
			return nil, fmt.Errorf("无效绕过网段 %s", value)
		}
		if ip.To4() == nil {
			return nil, fmt.Errorf("目前仅支持 IPv4 绕过网段 %s", value)
		}
		cidrs = append(cidrs, value)
	}
	if len(cidrs) == 0 {
		cidrs = []string{"127.0.0.0/8"}
	}
	return cidrs, nil
}

func BuildNFTables(config Config) (string, error) {
	cidrs, err := validate(config)
	if err != nil {
		return "", err
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
func (m *Manager) SupportedFor(config Config) bool {
	if config.Backend == "routeros_redirect" {
		_, err := exec.LookPath("iptables-legacy")
		return err == nil
	}
	return m.Supported()
}
func (m *Manager) ActiveFor(config Config) bool {
	if config.Backend == "routeros_redirect" {
		return command("iptables-legacy", "-t", "nat", "-C", "PREROUTING", "-p", "tcp", "-j", redirectChain) == nil
	}
	return m.Active()
}
func (m *Manager) Apply(config Config) error {
	if config.Backend == "routeros_redirect" {
		return m.applyRedirect(config)
	}
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
func (m *Manager) applyRedirect(config Config) error {
	cidrs, err := validate(config)
	if err != nil {
		return err
	}
	if !m.SupportedFor(config) {
		return fmt.Errorf("镜像缺少 iptables-legacy，无法启用 RouterOS TCP redirect")
	}
	// 只维护自己的链；不清空 RouterOS 或其他服务的防火墙规则。
	_ = command("iptables-legacy", "-t", "nat", "-N", redirectChain)
	if err := command("iptables-legacy", "-t", "nat", "-F", redirectChain); err != nil {
		return err
	}
	for _, cidr := range cidrs {
		if err := command("iptables-legacy", "-t", "nat", "-A", redirectChain, "-d", cidr, "-j", "RETURN"); err != nil {
			return err
		}
	}
	if err := command("iptables-legacy", "-t", "nat", "-A", redirectChain, "-p", "tcp", "-j", "REDIRECT", "--to-ports", strconv.Itoa(config.Port)); err != nil {
		return err
	}
	if !m.ActiveFor(config) {
		if err := command("iptables-legacy", "-t", "nat", "-A", "PREROUTING", "-p", "tcp", "-j", redirectChain); err != nil {
			return err
		}
	}
	return nil
}
func (m *Manager) DisableFor(config Config) error {
	if config.Backend != "routeros_redirect" {
		return m.Disable()
	}
	if !m.SupportedFor(config) {
		return nil
	}
	if m.ActiveFor(config) {
		if err := command("iptables-legacy", "-t", "nat", "-D", "PREROUTING", "-p", "tcp", "-j", redirectChain); err != nil {
			return err
		}
	}
	_ = command("iptables-legacy", "-t", "nat", "-F", redirectChain)
	_ = command("iptables-legacy", "-t", "nat", "-X", redirectChain)
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

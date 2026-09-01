package licensing

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const tokenPrefix = "CMS1"

var (
	ErrInvalidLicense = errors.New("授权码无效")
	ErrDeviceMismatch = errors.New("授权码与本设备不匹配")
	ErrLicenseExpired = errors.New("授权已过期")
	ErrNotConfigured  = errors.New("当前版本未配置授权公钥")
)

// Claims 是由授权签发工具签名、由管理端校验的授权内容。
type Claims struct {
	Version          int    `json:"version"`
	LicenseID        string `json:"license_id"`
	DeviceCode       string `json:"device_code"`
	MaxSubscriptions int    `json:"max_subscriptions"`
	IssuedAt         int64  `json:"issued_at"`
	ExpiresAt        int64  `json:"expires_at,omitempty"`
}

type Status struct {
	Valid            bool   `json:"is_valid"`
	Status           string `json:"status"`
	Reason           string `json:"reason,omitempty"`
	DeviceCode       string `json:"device_code"`
	LicenseID        string `json:"license_id,omitempty"`
	MaxSubscriptions int    `json:"max_subscriptions"`
	ExpiresAt        int64  `json:"expires_at,omitempty"`
}

type cachedLicense struct {
	LicenseCode string `json:"license_code"`
}

// Manager 管理本机设备标识及离线授权。它只使用 Go 标准库。
type Manager struct {
	mu         sync.RWMutex
	dataDir    string
	publicKey  ed25519.PublicKey
	deviceCode string
	freeLimit  int
	claims     *Claims
	reason     string
}

func NewManager(dataDir, publicKeyBase64 string, freeLimit int) (*Manager, error) {
	if freeLimit < 0 {
		return nil, fmt.Errorf("免费节点额度不能小于 0")
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("创建授权数据目录失败: %w", err)
	}
	deviceCode, err := loadOrCreateDeviceCode(filepath.Join(dataDir, "device.id"))
	if err != nil {
		return nil, err
	}
	m := &Manager{dataDir: dataDir, deviceCode: deviceCode, freeLimit: freeLimit}
	if strings.TrimSpace(publicKeyBase64) != "" {
		key, err := ParsePublicKey(publicKeyBase64)
		if err != nil {
			return nil, err
		}
		m.publicKey = key
	}
	m.loadCached()
	return m, nil
}

func ParsePublicKey(encoded string) (ed25519.PublicKey, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("授权公钥格式无效")
	}
	return ed25519.PublicKey(raw), nil
}

func ParsePrivateKey(encoded string) (ed25519.PrivateKey, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil || len(raw) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("授权私钥格式无效")
	}
	return ed25519.PrivateKey(raw), nil
}

func GenerateKey() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}

func Issue(privateKey ed25519.PrivateKey, claims Claims) (string, error) {
	if len(privateKey) != ed25519.PrivateKeySize || claims.DeviceCode == "" || claims.MaxSubscriptions < 1 {
		return "", fmt.Errorf("授权参数无效")
	}
	claims.Version = 1
	if claims.IssuedAt == 0 {
		claims.IssuedAt = time.Now().Unix()
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payloadText := base64.RawURLEncoding.EncodeToString(payload)
	message := tokenPrefix + "." + payloadText
	signature := ed25519.Sign(privateKey, []byte(message))
	return message + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func Verify(publicKey ed25519.PublicKey, token, deviceCode string, now time.Time) (Claims, error) {
	var claims Claims
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 || parts[0] != tokenPrefix || len(publicKey) != ed25519.PublicKeySize {
		return claims, ErrInvalidLicense
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, ErrInvalidLicense
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !ed25519.Verify(publicKey, []byte(parts[0]+"."+parts[1]), signature) {
		return claims, ErrInvalidLicense
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Version != 1 || claims.MaxSubscriptions < 1 {
		return claims, ErrInvalidLicense
	}
	if claims.DeviceCode != deviceCode {
		return claims, ErrDeviceMismatch
	}
	if claims.ExpiresAt > 0 && !now.Before(time.Unix(claims.ExpiresAt, 0)) {
		return claims, ErrLicenseExpired
	}
	return claims, nil
}

func (m *Manager) Activate(token string) (Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.publicKey) == 0 {
		return m.statusLocked(), ErrNotConfigured
	}
	claims, err := Verify(m.publicKey, token, m.deviceCode, time.Now())
	if err != nil {
		return m.statusLocked(), err
	}
	raw, err := json.MarshalIndent(cachedLicense{LicenseCode: strings.TrimSpace(token)}, "", "  ")
	if err != nil {
		return m.statusLocked(), err
	}
	if err := atomicWrite(filepath.Join(m.dataDir, "license.json"), raw, 0600); err != nil {
		return m.statusLocked(), err
	}
	m.claims = &claims
	m.reason = ""
	return m.statusLocked(), nil
}

func (m *Manager) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := os.Remove(filepath.Join(m.dataDir, "license.json")); err != nil && !os.IsNotExist(err) {
		return err
	}
	m.claims = nil
	m.reason = ""
	return nil
}

func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.claims != nil && m.claims.ExpiresAt > 0 && !time.Now().Before(time.Unix(m.claims.ExpiresAt, 0)) {
		m.claims = nil
		m.reason = ErrLicenseExpired.Error()
	}
	return m.statusLocked()
}

func (m *Manager) Limit() int { return m.Status().MaxSubscriptions }

func (m *Manager) statusLocked() Status {
	status := Status{Status: "unlicensed", Reason: m.reason, DeviceCode: m.deviceCode, MaxSubscriptions: m.freeLimit}
	if m.claims != nil {
		status.Valid = true
		status.Status = "licensed"
		status.Reason = ""
		status.LicenseID = m.claims.LicenseID
		status.MaxSubscriptions = m.claims.MaxSubscriptions
		status.ExpiresAt = m.claims.ExpiresAt
	}
	return status
}

func (m *Manager) loadCached() {
	raw, err := os.ReadFile(filepath.Join(m.dataDir, "license.json"))
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		m.reason = "读取本地授权失败"
		return
	}
	var cached cachedLicense
	if err := json.Unmarshal(raw, &cached); err != nil {
		m.reason = "本地授权文件损坏"
		return
	}
	if len(m.publicKey) == 0 {
		m.reason = ErrNotConfigured.Error()
		return
	}
	claims, err := Verify(m.publicKey, cached.LicenseCode, m.deviceCode, time.Now())
	if err != nil {
		m.reason = err.Error()
		return
	}
	m.claims = &claims
}

func loadOrCreateDeviceCode(path string) (string, error) {
	if raw, err := os.ReadFile(path); err == nil {
		id := strings.TrimSpace(string(raw))
		if len(id) >= 16 {
			return deviceCode(id), nil
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("读取设备标识失败: %w", err)
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("生成设备标识失败: %w", err)
	}
	id := base64.RawURLEncoding.EncodeToString(raw)
	if err := atomicWrite(path, []byte(id+"\n"), 0600); err != nil {
		return "", err
	}
	return deviceCode(id), nil
}

func deviceCode(id string) string {
	sum := sha256.Sum256([]byte(id))
	value := uint64(sum[0])<<40 | uint64(sum[1])<<32 | uint64(sum[2])<<24 | uint64(sum[3])<<16 | uint64(sum[4])<<8 | uint64(sum[5])
	return fmt.Sprintf("%06d", value%1000000)
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".license-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

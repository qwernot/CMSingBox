package api

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cmsingbox.local/cmsingbox/internal/storage"
)

func (s *Server) ensureBackHomeCertificate(settings *storage.Settings) error {
	if !settings.BackHomeEnabled || (settings.BackHomeCertPath != "" && settings.BackHomeKeyPath != "") {
		return nil
	}
	if strings.TrimSpace(settings.BackHomeServer) == "" || settings.BackHomePassword == "" {
		return fmt.Errorf("回家服务需要填写公网地址和密码")
	}
	certPath := filepath.Join(s.store.GetDataDir(), "backhome", "server.crt")
	keyPath := filepath.Join(s.store.GetDataDir(), "backhome", "server.key")
	regenerate := true
	if certPEM, err := os.ReadFile(certPath); err == nil {
		if block, _ := pem.Decode(certPEM); block != nil {
			if cert, parseErr := x509.ParseCertificate(block.Bytes); parseErr == nil && cert.VerifyHostname(settings.BackHomeServer) == nil && time.Now().Before(cert.NotAfter) {
				if _, keyErr := os.Stat(keyPath); keyErr == nil {
					regenerate = false
				}
			}
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if regenerate {
		if err := os.MkdirAll(filepath.Dir(certPath), 0700); err != nil {
			return err
		}
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return err
		}
		serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
		if err != nil {
			return err
		}
		template := &x509.Certificate{SerialNumber: serial, Subject: pkix.Name{CommonName: settings.BackHomeServer}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().AddDate(10, 0, 0), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
		if ip := net.ParseIP(settings.BackHomeServer); ip != nil {
			template.IPAddresses = []net.IP{ip}
		} else {
			template.DNSNames = []string{settings.BackHomeServer}
		}
		der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
		if err != nil {
			return err
		}
		keyDER, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			return err
		}
		if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0600); err != nil {
			return err
		}
		if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0644); err != nil {
			return err
		}
	}
	if _, err := os.Stat(keyPath); err != nil {
		return fmt.Errorf("回家证书私钥丢失: %w", err)
	}
	settings.BackHomeCertPath = certPath
	settings.BackHomeKeyPath = keyPath
	return nil
}

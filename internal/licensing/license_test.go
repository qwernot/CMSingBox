package licensing

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestIssueVerifyAndPersist(t *testing.T) {
	publicKey, privateKey, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	encodedPublic := base64.StdEncoding.EncodeToString(publicKey)
	dir := t.TempDir()
	manager, err := NewManager(dir, encodedPublic, 1)
	if err != nil {
		t.Fatal(err)
	}
	device := manager.Status().DeviceCode
	if len(device) != 6 || strings.Trim(device, "0123456789") != "" {
		t.Fatalf("unexpected device code %q", device)
	}
	token, err := Issue(privateKey, Claims{LicenseID: "customer-1", DeviceCode: device, MaxSubscriptions: 8})
	if err != nil {
		t.Fatal(err)
	}
	status, err := manager.Activate(token)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Valid || status.MaxSubscriptions != 8 {
		t.Fatalf("unexpected status: %#v", status)
	}
	reloaded, err := NewManager(dir, encodedPublic, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.Status(); !got.Valid || got.LicenseID != "customer-1" {
		t.Fatalf("cached license not loaded: %#v", got)
	}
}

func TestVerifyRejectsTamperDeviceAndExpiry(t *testing.T) {
	publicKey, privateKey, _ := GenerateKey()
	token, _ := Issue(privateKey, Claims{DeviceCode: "123456", MaxSubscriptions: 3})
	if _, err := Verify(publicKey, token+"x", "123456", time.Now()); !errors.Is(err, ErrInvalidLicense) {
		t.Fatalf("expected invalid license, got %v", err)
	}
	if _, err := Verify(publicKey, token, "654321", time.Now()); !errors.Is(err, ErrDeviceMismatch) {
		t.Fatalf("expected device mismatch, got %v", err)
	}
	expired, _ := Issue(privateKey, Claims{DeviceCode: "123456", MaxSubscriptions: 3, ExpiresAt: time.Now().Add(-time.Minute).Unix()})
	if _, err := Verify(publicKey, expired, "123456", time.Now()); !errors.Is(err, ErrLicenseExpired) {
		t.Fatalf("expected expired license, got %v", err)
	}
}

func TestNoPublicKeyUsesFreeLimit(t *testing.T) {
	manager, err := NewManager(t.TempDir(), "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if status := manager.Status(); status.Valid || status.MaxSubscriptions != 1 {
		t.Fatalf("unexpected free status: %#v", status)
	}
	if _, err := manager.Activate("anything"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected not configured, got %v", err)
	}
}

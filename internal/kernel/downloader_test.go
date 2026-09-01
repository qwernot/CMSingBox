package kernel

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMirroredKernelSHA256CoversSupportedArchitectures(t *testing.T) {
	for _, version := range []string{"v1.13.21", "v1.14.0"} {
		for _, arch := range []string{"amd64", "arm64", "arm"} {
			if got := mirroredKernelSHA256(version, arch); len(got) != 64 {
				t.Fatalf("mirroredKernelSHA256(%q, %q) = %q", version, arch, got)
			}
		}
	}
}

func TestVerifySHA256(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kernel")
	if err := os.WriteFile(path, []byte("cmsingbox"), 0o600); err != nil {
		t.Fatal(err)
	}
	const expected = "fd91ad25ef088e16b8d83708754971f914b14829797d1b5f9b7b89d14428ee45"
	if err := verifySHA256(path, expected); err != nil {
		t.Fatalf("verifySHA256() error = %v", err)
	}
	if err := verifySHA256(path, "0000000000000000000000000000000000000000000000000000000000000000"); err == nil {
		t.Fatal("verifySHA256() accepted an invalid checksum")
	}
}

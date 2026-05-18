package autotls

import (
	"crypto/ecdsa"
	"crypto/x509"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadOrGenerateCreatesNewCert(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")

	cert, err := LoadOrGenerate(nil, []string{"localhost"}, certFile, keyFile)
	if err != nil {
		t.Fatalf("LoadOrGenerate: %v", err)
	}
	if cert.PrivateKey == nil {
		t.Error("expected non-nil private key")
	}
	if cert.Leaf == nil {
		t.Fatal("expected non-nil leaf certificate")
	}
	if len(cert.Leaf.Subject.Organization) == 0 || cert.Leaf.Subject.Organization[0] != "Edge Dispatch AutoTLS" {
		t.Errorf("org = %v, want [\"Edge Dispatch AutoTLS\"]", cert.Leaf.Subject.Organization)
	}

	_, ok := cert.PrivateKey.(*ecdsa.PrivateKey)
	if !ok {
		t.Errorf("private key type = %T, want *ecdsa.PrivateKey", cert.PrivateKey)
	}

	if _, err := os.Stat(certFile); err != nil {
		t.Errorf("cert file not written: %v", err)
	}
	if _, err := os.Stat(keyFile); err != nil {
		t.Errorf("key file not written: %v", err)
	}
}

func TestLoadOrGenerateLoadsExistingCert(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")

	cert1, err := LoadOrGenerate(nil, nil, certFile, keyFile)
	if err != nil {
		t.Fatalf("first LoadOrGenerate: %v", err)
	}

	cert2, err := LoadOrGenerate(nil, nil, certFile, keyFile)
	if err != nil {
		t.Fatalf("second LoadOrGenerate: %v", err)
	}

	if cert1.Leaf.SerialNumber.Cmp(cert2.Leaf.SerialNumber) != 0 {
		t.Error("serial numbers differ between loads, expected same cert loaded from disk")
	}
}

func TestLoadOrGenerateWithSANs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")

	ips := []net.IP{net.ParseIP("10.0.0.1")}
	dnsNames := []string{"edge.local", "node.example.com"}

	cert, err := LoadOrGenerate(ips, dnsNames, certFile, keyFile)
	if err != nil {
		t.Fatalf("LoadOrGenerate: %v", err)
	}

	foundIP, foundDNS1, foundDNS2 := false, false, false
	for _, ip := range cert.Leaf.IPAddresses {
		if ip.Equal(ips[0]) {
			foundIP = true
		}
	}
	for _, dns := range cert.Leaf.DNSNames {
		switch dns {
		case "edge.local":
			foundDNS1 = true
		case "node.example.com":
			foundDNS2 = true
		}
	}
	if !foundIP {
		t.Error("expected SAN IP 10.0.0.1")
	}
	if !foundDNS1 {
		t.Error("expected SAN DNS edge.local")
	}
	if !foundDNS2 {
		t.Error("expected SAN DNS node.example.com")
	}
}

func TestLoadOrGenerateValidCert(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")

	dnsNames := []string{"localhost", "test.local"}
	cert, err := LoadOrGenerate(nil, dnsNames, certFile, keyFile)
	if err != nil {
		t.Fatalf("LoadOrGenerate: %v", err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(cert.Leaf)
	opts := x509.VerifyOptions{DNSName: "localhost", Roots: roots}
	opts.KeyUsages = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	if _, err := cert.Leaf.Verify(opts); err != nil {
		t.Errorf("self-signed cert failed verification: %v", err)
	}
}

func TestLoadOrGenerateFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permissions are not enforced on Windows")
	}
	t.Parallel()
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")

	_, err := LoadOrGenerate(nil, nil, certFile, keyFile)
	if err != nil {
		t.Fatalf("LoadOrGenerate: %v", err)
	}

	for _, f := range []string{certFile, keyFile} {
		info, err := os.Stat(f)
		if err != nil {
			t.Fatalf("stat %s: %v", f, err)
		}
		if mode := info.Mode().Perm(); mode != 0600 {
			t.Errorf("%s permissions = %04o, want 0600", f, mode)
		}
	}
}

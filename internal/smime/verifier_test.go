package smime

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"path/filepath"
	"testing"
	"time"

	"github.com/hkdb/aerion/internal/database"
	"github.com/rs/zerolog"
	"go.mozilla.org/pkcs7"
)

type testCert struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
}

var testSerial int64

// newTestCert creates a certificate signed by parent, or self-signed when
// parent is nil. CA certificates get no email address.
func newTestCert(t *testing.T, cn, email string, isCA bool, parent *testCert) *testCert {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	testSerial++
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(testSerial),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		BasicConstraintsValid: true,
		IsCA:                  isCA,
	}
	if isCA {
		tmpl.KeyUsage = x509.KeyUsageCertSign
	} else {
		tmpl.KeyUsage = x509.KeyUsageDigitalSignature
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageEmailProtection}
		tmpl.EmailAddresses = []string{email}
	}
	issuer, issuerKey := tmpl, key
	if parent != nil {
		issuer, issuerKey = parent.cert, parent.key
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, issuer, &key.PublicKey, issuerKey)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return &testCert{cert: cert, key: key}
}

// signTestMessage returns a parsed PKCS#7 object signed by signer. Extra
// certificates are embedded before the signer's certificate.
func signTestMessage(t *testing.T, signer *testCert, extra ...*x509.Certificate) *pkcs7.PKCS7 {
	t.Helper()
	sd, err := pkcs7.NewSignedData([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	sd.SetDigestAlgorithm(pkcs7.OIDDigestAlgorithmSHA256)
	for _, c := range extra {
		sd.AddCertificate(c)
	}
	if err := sd.AddSigner(signer.cert, signer.key, pkcs7.SignerInfoConfig{}); err != nil {
		t.Fatal(err)
	}
	der, err := sd.Finish()
	if err != nil {
		t.Fatal(err)
	}
	p7, err := pkcs7.Parse(der)
	if err != nil {
		t.Fatal(err)
	}
	return p7
}

func newTestVerifier(t *testing.T, roots ...*x509.Certificate) (*Verifier, *Store) {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	store := NewStore(db.DB, zerolog.Nop())
	v := NewVerifier(store, zerolog.Nop())
	v.roots = x509.NewCertPool()
	for _, r := range roots {
		v.roots.AddCert(r)
	}
	return v, store
}

func TestVerifyPKCS7TrustStatus(t *testing.T) {
	trustedCA := newTestCert(t, "Trusted CA", "", true, nil)
	untrustedCA := newTestCert(t, "Attacker CA", "", true, nil)

	tests := []struct {
		name   string
		signer *testCert
		extra  []*x509.Certificate
		want   SignatureStatus
	}{
		{"trusted chain", newTestCert(t, "Alice", "alice@example.com", false, trustedCA), nil, StatusSigned},
		{"trusted chain with intermediate embedded", newTestCert(t, "Alice", "alice@example.com", false, trustedCA), []*x509.Certificate{trustedCA.cert}, StatusSigned},
		{"untrusted CA", newTestCert(t, "Mallory", "alice@example.com", false, untrustedCA), []*x509.Certificate{untrustedCA.cert}, StatusUnknownSigner},
		{"self-signed", newTestCert(t, "Alice", "alice@example.com", false, nil), nil, StatusSelfSigned},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, _ := newTestVerifier(t, trustedCA.cert)
			result := v.verifyPKCS7(signTestMessage(t, tt.signer, tt.extra...))
			if result.Status != tt.want {
				t.Fatalf("status = %q (%s), want %q", result.Status, result.ErrorMessage, tt.want)
			}
		})
	}
}

func TestVerifyPKCS7UsesActualSignerCert(t *testing.T) {
	ca := newTestCert(t, "Trusted CA", "", true, nil)
	signer := newTestCert(t, "Mallory", "mallory@example.com", false, ca)
	decoy := newTestCert(t, "Alice", "alice@example.com", false, ca)

	v, store := newTestVerifier(t, ca.cert)
	result := v.verifyPKCS7(signTestMessage(t, signer, decoy.cert))

	if result.Status != StatusSigned {
		t.Fatalf("status = %q (%s), want %q", result.Status, result.ErrorMessage, StatusSigned)
	}
	if result.SignerEmail != "mallory@example.com" || result.SignerName != "Mallory" {
		t.Errorf("signer = %q <%s>, want Mallory <mallory@example.com>", result.SignerName, result.SignerEmail)
	}

	if certs, err := store.GetSenderCerts("alice@example.com"); err != nil || len(certs) != 0 {
		t.Errorf("decoy cached for alice: %d certs, err %v", len(certs), err)
	}
	certs, err := store.GetSenderCerts("mallory@example.com")
	if err != nil || len(certs) != 1 {
		t.Fatalf("mallory certs = %d, err %v; want 1", len(certs), err)
	}
	if certs[0].SerialNumber != signer.cert.SerialNumber.String() {
		t.Errorf("cached serial %s, want signer serial %s", certs[0].SerialNumber, signer.cert.SerialNumber)
	}
}

func TestVerifyPKCS7ChecksEverySigner(t *testing.T) {
	trustedCA := newTestCert(t, "Trusted CA", "", true, nil)
	untrustedCA := newTestCert(t, "Attacker CA", "", true, nil)
	alice := newTestCert(t, "Alice", "alice@example.com", false, trustedCA)
	mallory := newTestCert(t, "Mallory", "mallory@example.com", false, untrustedCA)

	sd, err := pkcs7.NewSignedData([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	sd.SetDigestAlgorithm(pkcs7.OIDDigestAlgorithmSHA256)
	for _, s := range []*testCert{alice, mallory} {
		if err := sd.AddSigner(s.cert, s.key, pkcs7.SignerInfoConfig{}); err != nil {
			t.Fatal(err)
		}
	}
	der, err := sd.Finish()
	if err != nil {
		t.Fatal(err)
	}
	p7, err := pkcs7.Parse(der)
	if err != nil {
		t.Fatal(err)
	}

	v, _ := newTestVerifier(t, trustedCA.cert)
	if result := v.verifyPKCS7(p7); result.Status != StatusUnknownSigner {
		t.Fatalf("status = %q (%s), want %q", result.Status, result.ErrorMessage, StatusUnknownSigner)
	}
}

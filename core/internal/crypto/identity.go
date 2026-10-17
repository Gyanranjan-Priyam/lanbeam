// Package crypto manages device identity: generating/loading ECDSA P-256
// key pairs, self-signed certificates, and computing SHA-256 fingerprints.
package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// Identity holds the device's TLS certificate and its computed fingerprint.
type Identity struct {
	// Cert is the TLS certificate (private key + DER certificate).
	Cert tls.Certificate

	// Fingerprint is the lowercase hex SHA-256 of the raw DER certificate.
	Fingerprint string
}

// New generates a new ECDSA P-256 identity.
// It does NOT persist to disk; use LoadOrCreate for persistence.
func New() (*Identity, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return newFromKey(key)
}

// LoadOrCreate loads the identity from dir, or generates and saves a new one.
// Files written: <dir>/identity.key (PEM, mode 0600) and <dir>/identity.crt (PEM).
func LoadOrCreate(dir string) (*Identity, error) {
	keyPath := filepath.Join(dir, "identity.key")
	crtPath := filepath.Join(dir, "identity.crt")

	certPEM, errCrt := os.ReadFile(crtPath)
	keyPEM, errKey := os.ReadFile(keyPath)

	if errCrt == nil && errKey == nil {
		// Both files exist: parse and return.
		cert, err := tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			return nil, err
		}
		fp, err := fingerprintFromCert(cert)
		if err != nil {
			return nil, err
		}
		return &Identity{Cert: cert, Fingerprint: fp}, nil
	}

	// Generate new identity.
	id, err := New()
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}

	// Serialize key.
	keyDER, err := x509.MarshalECPrivateKey(id.Cert.PrivateKey.(*ecdsa.PrivateKey))
	if err != nil {
		return nil, err
	}
	keyBlock := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(keyPath, keyBlock, 0o600); err != nil {
		return nil, err
	}

	// Serialize certificate.
	crtBlock := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: id.Cert.Certificate[0]})
	if err := os.WriteFile(crtPath, crtBlock, 0o644); err != nil {
		return nil, err
	}

	return id, nil
}

// PinnedTLSConfig returns a *tls.Config that accepts only the certificate
// whose fingerprint matches expectedFP (lowercase hex SHA-256).
//
// InsecureSkipVerify is intentional: self-signed certs cannot pass normal CA
// verification. The fingerprint check in VerifyPeerCertificate replaces it.
func PinnedTLSConfig(expectedFP string) *tls.Config {
	return &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true, //nolint:gosec // See doc above.
		VerifyPeerCertificate: func(raw [][]byte, _ [][]*x509.Certificate) error {
			if len(raw) == 0 {
				return errors.New("crypto: no peer certificate")
			}
			sum := sha256.Sum256(raw[0])
			got := hex.EncodeToString(sum[:])
			if got != expectedFP {
				return errors.New("crypto: peer fingerprint mismatch")
			}
			return nil
		},
	}
}

// TrustFirstTLSConfig returns a *tls.Config that captures the remote fingerprint
// on first use. The caller must validate fp with the user before trusting it.
func TrustFirstTLSConfig(out *string) *tls.Config {
	return &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true, //nolint:gosec // Fingerprint returned to caller for user confirmation.
		VerifyPeerCertificate: func(raw [][]byte, _ [][]*x509.Certificate) error {
			if len(raw) == 0 {
				return errors.New("crypto: no peer certificate")
			}
			sum := sha256.Sum256(raw[0])
			*out = hex.EncodeToString(sum[:])
			return nil
		},
	}
}

// newFromKey builds an Identity from an existing private key.
func newFromKey(key *ecdsa.PrivateKey) (*Identity, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	if err != nil {
		return nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "lanbeam"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(der)
	cert := tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  key,
	}
	return &Identity{
		Cert:        cert,
		Fingerprint: hex.EncodeToString(sum[:]),
	}, nil
}

// fingerprintFromCert extracts the fingerprint from a parsed tls.Certificate.
func fingerprintFromCert(cert tls.Certificate) (string, error) {
	if len(cert.Certificate) == 0 {
		return "", errors.New("crypto: certificate has no DER data")
	}
	sum := sha256.Sum256(cert.Certificate[0])
	return hex.EncodeToString(sum[:]), nil
}

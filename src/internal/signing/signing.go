// Package signing implements publisher provenance for mod release
// artifacts: an Ed25519 signature over sha256(zip)||sha256(manifest.json).
// This is a trust anchor independent of the GitHub account that hosts the
// release — sha256 alone only proves transport integrity (these are the
// bytes that arrived), never authorship (these are the bytes the real
// publisher intended). The private key never touches GitHub, so a
// compromised repo/account cannot produce a validly-signed release.
package signing

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// GenerateKey creates a new Ed25519 key pair for a publisher.
func GenerateKey() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}

// digest is the message actually signed: sha256(zip) || sha256(manifest),
// concatenated as raw bytes (32+32), not hex-encoded — the signature covers
// both the release artifact and the manifest that describes it, so a
// publisher can't swap in a different manifest after signing.
func digest(zipBytes, manifestBytes []byte) []byte {
	zipSum := sha256.Sum256(zipBytes)
	manifestSum := sha256.Sum256(manifestBytes)
	out := make([]byte, 0, len(zipSum)+len(manifestSum))
	out = append(out, zipSum[:]...)
	out = append(out, manifestSum[:]...)
	return out
}

// Sign returns a base64-encoded Ed25519 signature over digest(zip, manifest).
func Sign(priv ed25519.PrivateKey, zipBytes, manifestBytes []byte) (string, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("invalid private key size: got %d bytes, want %d", len(priv), ed25519.PrivateKeySize)
	}
	sig := ed25519.Sign(priv, digest(zipBytes, manifestBytes))
	return base64.StdEncoding.EncodeToString(sig), nil
}

// Verify reports whether signatureB64 is a valid signature over
// digest(zip, manifest) under pub. A malformed signature (not valid
// base64, or wrong length) is an error, not a silent false result — the
// caller should treat both an error and ok=false as "reject".
func Verify(pub ed25519.PublicKey, zipBytes, manifestBytes []byte, signatureB64 string) (bool, error) {
	sig, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return false, fmt.Errorf("signature is not valid base64: %w", err)
	}
	if len(sig) != ed25519.SignatureSize {
		return false, fmt.Errorf("invalid signature size: got %d bytes, want %d", len(sig), ed25519.SignatureSize)
	}
	if len(pub) != ed25519.PublicKeySize {
		return false, fmt.Errorf("invalid public key size: got %d bytes, want %d", len(pub), ed25519.PublicKeySize)
	}
	return ed25519.Verify(pub, digest(zipBytes, manifestBytes), sig), nil
}

// EncodePublicKey/EncodePrivateKey/DecodePublicKey/DecodePrivateKey give
// keys a plain base64 text form for storing in a file or a database column
// — no PEM/ASN.1 envelope, Ed25519 keys are already fixed-size raw bytes.

func EncodePublicKey(pub ed25519.PublicKey) string {
	return base64.StdEncoding.EncodeToString(pub)
}

func EncodePrivateKey(priv ed25519.PrivateKey) string {
	return base64.StdEncoding.EncodeToString(priv)
}

func DecodePublicKey(s string) (ed25519.PublicKey, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("public key is not valid base64: %w", err)
	}
	if len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: got %d bytes, want %d", len(b), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(b), nil
}

func DecodePrivateKey(s string) (ed25519.PrivateKey, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("private key is not valid base64: %w", err)
	}
	if len(b) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size: got %d bytes, want %d", len(b), ed25519.PrivateKeySize)
	}
	return ed25519.PrivateKey(b), nil
}

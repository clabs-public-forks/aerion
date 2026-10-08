package crypto

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestNewEncryptor(t *testing.T) {
	dir := t.TempDir()
	enc, err := NewEncryptor(dir)
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}
	if enc == nil {
		t.Fatal("expected non-nil encryptor")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	enc, err := NewEncryptor(t.TempDir())
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}

	plaintext := "hello world"
	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if ciphertext == "" {
		t.Fatal("expected non-empty ciphertext")
	}
	if ciphertext == plaintext {
		t.Fatal("ciphertext should differ from plaintext")
	}

	decrypted, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestEncryptDecryptEmpty(t *testing.T) {
	enc, err := NewEncryptor(t.TempDir())
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}

	ciphertext, err := enc.Encrypt("")
	if err != nil {
		t.Fatalf("Encrypt empty failed: %v", err)
	}
	if ciphertext != "" {
		t.Fatalf("expected empty ciphertext, got %q", ciphertext)
	}

	decrypted, err := enc.Decrypt("")
	if err != nil {
		t.Fatalf("Decrypt empty failed: %v", err)
	}
	if decrypted != "" {
		t.Fatalf("expected empty decrypted, got %q", decrypted)
	}
}

func TestDecryptInvalidBase64(t *testing.T) {
	enc, err := NewEncryptor(t.TempDir())
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}

	_, err = enc.Decrypt("not-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	enc, err := NewEncryptor(t.TempDir())
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}

	ciphertext, err := enc.Encrypt("secret data")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Decode, tamper, re-encode
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		t.Fatalf("base64 decode failed: %v", err)
	}
	// Flip a byte near the end (in the ciphertext portion, not the nonce)
	data[len(data)-1] ^= 0xFF
	tampered := base64.StdEncoding.EncodeToString(data)

	_, err = enc.Decrypt(tampered)
	if err == nil {
		t.Fatal("expected error for tampered ciphertext")
	}
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	enc, err := NewEncryptor(t.TempDir())
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}

	plaintext := "same input"
	ct1, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("first Encrypt failed: %v", err)
	}
	ct2, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("second Encrypt failed: %v", err)
	}

	if ct1 == ct2 {
		t.Fatal("expected different ciphertexts due to random nonce")
	}
}

func TestLoadOrCreateKey(t *testing.T) {
	t.Run("uses stored key bytes, falls back to derived key", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, keyFileName)
		salt := bytes.Repeat([]byte{1}, saltSize)
		stored := bytes.Repeat([]byte{2}, keySize) // differs from deriveKey(salt), as after a hostname change
		if err := os.WriteFile(path, append(append([]byte{}, salt...), stored...), 0600); err != nil {
			t.Fatal(err)
		}

		// A value encrypted under the derived key (older builds, new hostname).
		legacy := &Encryptor{key: deriveKey(salt)}
		legacyCT, err := legacy.Encrypt("legacy")
		if err != nil {
			t.Fatal(err)
		}

		e, err := NewEncryptor(dir)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(e.key, stored) {
			t.Fatal("stored key bytes not used")
		}
		ct, _ := e.Encrypt("current")
		if got, err := e.Decrypt(ct); err != nil || got != "current" {
			t.Fatalf("current: got %q, %v", got, err)
		}
		if got, err := e.Decrypt(legacyCT); err != nil || got != "legacy" {
			t.Fatalf("legacy: got %q, %v", got, err)
		}
	})

	t.Run("damaged key file is moved aside, not overwritten", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, keyFileName)
		if err := os.WriteFile(path, []byte("short"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewEncryptor(dir); err != nil {
			t.Fatal(err)
		}
		matches, _ := filepath.Glob(path + ".bad-*")
		if len(matches) != 1 {
			t.Fatalf("expected one moved-aside file, got %v", matches)
		}
		if data, _ := os.ReadFile(matches[0]); string(data) != "short" {
			t.Fatal("moved-aside file content changed")
		}
	})
}

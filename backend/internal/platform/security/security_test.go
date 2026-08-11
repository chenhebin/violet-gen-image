package security

import "testing"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()
	plaintext := []byte("sk-sensitive-value")
	ciphertext, err := Encrypt(plaintext, "test-encryption-key-with-at-least-32-bytes")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if string(ciphertext) == string(plaintext) {
		t.Fatal("Encrypt() returned plaintext")
	}
	decrypted, err := Decrypt(ciphertext, "test-encryption-key-with-at-least-32-bytes")
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("Decrypt() = %q, want %q", decrypted, plaintext)
	}
}

func TestDigestAndNormalization(t *testing.T) {
	t.Parallel()
	first := HMACDigest("YY-CODE", "pepper")
	second := HMACDigest("YY-CODE", "pepper")
	if first != second {
		t.Fatal("HMACDigest() is not deterministic")
	}
	if first == HMACDigest("YY-CODE-2", "pepper") {
		t.Fatal("HMACDigest() ignored input")
	}
	if got := NormalizeEmail(" Demo@Yingyan.Local "); got != "demo@yingyan.local" {
		t.Fatalf("NormalizeEmail() = %q", got)
	}
	if got := NormalizeRedemptionCode(" yy-start-10 "); got != "YY-START-10" {
		t.Fatalf("NormalizeRedemptionCode() = %q", got)
	}
}

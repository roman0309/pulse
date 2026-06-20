package secrets

import "testing"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	b := New("primary-key")
	enc, err := b.Encrypt("ssh-private-key")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if enc == "ssh-private-key" {
		t.Fatal("ciphertext equals plaintext")
	}
	got, err := b.Decrypt(enc)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != "ssh-private-key" {
		t.Fatalf("got %q", got)
	}
}

func TestDecryptFallsBackToLegacyKey(t *testing.T) {
	// Data written under the old key...
	old := New("legacy-key")
	enc, err := old.Encrypt("secret")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	// ...still opens when a new primary key keeps the legacy one as fallback.
	rotated := New("new-primary", "legacy-key")
	got, err := rotated.Decrypt(enc)
	if err != nil {
		t.Fatalf("legacy decrypt failed: %v", err)
	}
	if got != "secret" {
		t.Fatalf("got %q", got)
	}
}

func TestDecryptFailsWithoutMatchingKey(t *testing.T) {
	enc, _ := New("legacy-key").Encrypt("secret")
	if _, err := New("unrelated-key").Decrypt(enc); err == nil {
		t.Fatal("expected decrypt to fail without the matching key")
	}
}

func TestEncryptUsesPrimaryKey(t *testing.T) {
	// A box that only knows the new key must read back what the rotated box wrote.
	rotated := New("new-primary", "legacy-key")
	enc, err := rotated.Encrypt("fresh")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	got, err := New("new-primary").Decrypt(enc)
	if err != nil {
		t.Fatalf("primary decrypt failed: %v", err)
	}
	if got != "fresh" {
		t.Fatalf("got %q", got)
	}
}

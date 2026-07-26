package security

import (
	"bytes"
	"strings"
	"testing"
	"unicode"

	"golang.org/x/crypto/scrypt"
)

// TestScryptRFC7914Vector confirms the scrypt KDF matches the RFC 7914 vector,
// which is the same standard the former Python hashlib.scrypt uses — so hashes
// produced by either implementation verify in the other.
func TestScryptRFC7914Vector(t *testing.T) {
	got, err := scrypt.Key([]byte(""), []byte(""), 16, 1, 1, 64)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{
		0x77, 0xd6, 0x57, 0x62, 0x38, 0x65, 0x7b, 0x20, 0x3b, 0x19, 0xca, 0x42, 0xc1, 0x8a, 0x04, 0x97,
		0xf1, 0x6b, 0x48, 0x44, 0xe3, 0x07, 0x4a, 0xe8, 0xdf, 0xdf, 0xfa, 0x3f, 0xed, 0xe2, 0x14, 0x42,
		0xfc, 0xd0, 0x06, 0x9d, 0xed, 0x09, 0x48, 0xf8, 0x32, 0x6a, 0x75, 0x3a, 0x0f, 0xc8, 0x1f, 0x17,
		0xe8, 0xd3, 0xe0, 0xfb, 0x2e, 0x0d, 0x36, 0x28, 0xcf, 0x35, 0xe2, 0x0c, 0x38, 0xd1, 0x89, 0x06,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("scrypt RFC 7914 vector mismatch")
	}
}

func TestHashVerifyRoundTrip(t *testing.T) {
	hash, err := HashPassword("Sup3rSecret!")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "scrypt$16384$8$1$") {
		t.Fatalf("unexpected format: %s", hash)
	}
	if !VerifyPassword("Sup3rSecret!", hash) {
		t.Fatal("correct password failed to verify")
	}
	if VerifyPassword("wrong", hash) {
		t.Fatal("wrong password verified")
	}
	if VerifyPassword("Sup3rSecret!", "scrypt$16384$8$1$bad$data") {
		t.Fatal("malformed hash verified")
	}
	if VerifyPassword("x", "not-a-hash") {
		t.Fatal("non-hash verified")
	}
}

func TestRandomCredential(t *testing.T) {
	for range 50 {
		c := RandomCredential(12)
		if len(c) != 12 {
			t.Fatalf("length = %d", len(c))
		}
		if !unicode.IsLetter(rune(c[0])) {
			t.Fatalf("first char not a letter: %q", c)
		}
		if strings.IndexFunc(c, unicode.IsLower) < 0 || strings.IndexFunc(c, unicode.IsUpper) < 0 || strings.IndexFunc(c, unicode.IsDigit) < 0 {
			t.Fatalf("missing character class: %q", c)
		}
	}
}

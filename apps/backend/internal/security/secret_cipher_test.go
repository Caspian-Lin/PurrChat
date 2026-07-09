package security

import (
	"strings"
	"testing"
)

// genKey 生成 base64 编码的 32 字节密钥用于测试
func genKey(t *testing.T) string {
	t.Helper()
	// 32 字节全 0 的 base64 编码（合法 256-bit 密钥）
	return "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	c, err := NewCipherFromKey(genKey(t))
	if err != nil {
		t.Fatalf("NewCipherFromKey: %v", err)
	}

	cases := []string{
		"",
		"sk-xxxxxxxxxxxxxxxxxxxx",
		"https://api.openai.com/v1/chat/completions",
		"中文密钥内容 🔑",
		strings.Repeat("a", 4096),
	}

	for _, pt := range cases {
		ct, err := c.Encrypt(pt)
		if err != nil {
			t.Fatalf("Encrypt(%q): %v", pt, err)
		}
		got, err := c.Decrypt(ct)
		if err != nil {
			t.Fatalf("Decrypt: %v", err)
		}
		if got != pt {
			t.Errorf("round-trip mismatch: got %q, want %q", got, pt)
		}
	}
}

func TestDifferentIVProducesDifferentCiphertext(t *testing.T) {
	c, err := NewCipherFromKey(genKey(t))
	if err != nil {
		t.Fatalf("NewCipherFromKey: %v", err)
	}

	pt := "same-plaintext"
	a, _ := c.Encrypt(pt)
	b, _ := c.Encrypt(pt)
	if a == b {
		t.Error("expected different ciphertexts due to random IV, got identical")
	}

	// 两者都应能解出相同明文
	if d, _ := c.Decrypt(a); d != pt {
		t.Errorf("decrypt A = %q", d)
	}
	if d, _ := c.Decrypt(b); d != pt {
		t.Errorf("decrypt B = %q", d)
	}
}

func TestEmptyKeyErrors(t *testing.T) {
	_, err := NewCipherFromKey("")
	if err != ErrMasterKeyNotSet {
		t.Errorf("expected ErrMasterKeyNotSet, got %v", err)
	}
}

func TestInvalidKeyLengthErrors(t *testing.T) {
	// 有效 base64，但只有 16 字节
	short := "AAAAAAAAAAAAAAAAAAAAAA=="
	_, err := NewCipherFromKey(short)
	if err == nil {
		t.Fatal("expected error for short key, got nil")
	}
}

func TestDecryptTamperedCiphertextFails(t *testing.T) {
	c, err := NewCipherFromKey(genKey(t))
	if err != nil {
		t.Fatalf("NewCipherFromKey: %v", err)
	}
	ct, _ := c.Encrypt("secret")

	// 篡改最后几个字符
	tampered := ct[:len(ct)-4] + "XXXX"
	if _, err := c.Decrypt(tampered); err == nil {
		t.Error("expected decryption failure for tampered ciphertext")
	}
}

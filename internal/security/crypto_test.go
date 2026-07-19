package security

import "testing"

func TestWrapUnwrapPINAndRecovery(t *testing.T) {
	dek, err := GenerateDEK()
	if err != nil {
		t.Fatal(err)
	}
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatal(err)
	}
	pin := "123456"
	wrappedPIN, err := WrapDEKWithPIN(pin, salt, dek)
	if err != nil {
		t.Fatal(err)
	}
	unwrapped, err := UnwrapDEKWithPIN(pin, salt, wrappedPIN, argon2Time, argon2MemoryKiB, argon2Threads)
	if err != nil {
		t.Fatal(err)
	}
	if string(unwrapped) != string(dek) {
		t.Fatalf("PIN unwrap mismatch")
	}

	_, recoveryRaw, err := GenerateRecoveryKey()
	if err != nil {
		t.Fatal(err)
	}
	wrappedRecovery, err := WrapDEKWithRecovery(recoveryRaw, dek)
	if err != nil {
		t.Fatal(err)
	}
	unwrappedRecovery, err := UnwrapDEKWithRecovery(recoveryRaw, wrappedRecovery)
	if err != nil {
		t.Fatal(err)
	}
	if string(unwrappedRecovery) != string(dek) {
		t.Fatalf("recovery unwrap mismatch")
	}

	encrypted, err := EncryptBytes(dek, []byte("portfolio-bytes"))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := DecryptBytes(dek, encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if string(plain) != "portfolio-bytes" {
		t.Fatalf("encrypt roundtrip failed")
	}
}

package tests

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// Verifies bcrypt round-trip and salt randomness (different hashes for same input).
func TestPasswordHashAndCompare(t *testing.T) {
	pw := "Password123!"
	h1, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	h2, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if string(h1) == string(h2) {
		t.Fatal("bcrypt should return different hashes due to random salt")
	}
	if err := bcrypt.CompareHashAndPassword(h1, []byte(pw)); err != nil {
		t.Fatalf("compare same pw: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword(h1, []byte("wrong")); err == nil {
		t.Fatal("compare wrong pw should fail")
	}
}

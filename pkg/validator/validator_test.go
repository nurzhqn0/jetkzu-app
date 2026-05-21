package validator

import "testing"

func TestEmail(t *testing.T) {
	if err := Email("foo@bar.kz"); err != nil {
		t.Fatalf("good email rejected: %v", err)
	}
	if err := Email("not-an-email"); err == nil {
		t.Fatal("bad email accepted")
	}
}

func TestOneOf(t *testing.T) {
	if err := OneOf("role", "passenger", "passenger", "driver"); err != nil {
		t.Fatalf("valid value rejected: %v", err)
	}
	if err := OneOf("role", "alien", "passenger", "driver"); err == nil {
		t.Fatal("invalid value accepted")
	}
}

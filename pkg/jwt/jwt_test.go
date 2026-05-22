package jwt

import (
	"testing"
	"time"
)

func TestJWTIssueAndParse(t *testing.T) {
	m := New("test-secret", time.Hour)
	token, exp, err := m.Issue("user-1", "user@example.com", "passenger")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if token == "" || time.Until(exp) <= 0 {
		t.Fatalf("bad token or expiry: %v %v", token, exp)
	}
	claims, err := m.Parse(token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.UserID != "user-1" || claims.Role != "passenger" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestJWTRejectsBadSecret(t *testing.T) {
	m := New("secret-a", time.Hour)
	token, _, _ := m.Issue("user-1", "u@e.kz", "passenger")

	m2 := New("secret-b", time.Hour)
	if _, err := m2.Parse(token); err == nil {
		t.Fatal("expected parse error with wrong secret")
	}
}

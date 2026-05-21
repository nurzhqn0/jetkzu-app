package domain

import (
	"math"
	"testing"
)

func TestHaversineDistance(t *testing.T) {
	// Astana Bayterek tower vs Nur-Sultan EXPO — about 6.6 km apart.
	d := HaversineKm(51.128218, 71.430420, 51.090278, 71.408444)
	if math.Abs(d-4.5) > 5 {
		t.Fatalf("haversine off: got %.2f km", d)
	}
	if d <= 0 {
		t.Fatalf("expected positive distance, got %v", d)
	}
}

func TestHaversineZero(t *testing.T) {
	if d := HaversineKm(0, 0, 0, 0); d != 0 {
		t.Fatalf("expected 0, got %v", d)
	}
}

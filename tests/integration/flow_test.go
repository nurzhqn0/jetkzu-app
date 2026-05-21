package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

// Integration test runs only when RUN_INTEGRATION=1.
// It assumes `docker compose up` is already running.
//
// Run with:
//   RUN_INTEGRATION=1 GATEWAY_URL=http://localhost:8080 go test ./tests/integration/...

func skipIfDisabled(t *testing.T) string {
	t.Helper()
	if os.Getenv("RUN_INTEGRATION") != "1" {
		t.Skip("set RUN_INTEGRATION=1 to run")
	}
	url := os.Getenv("GATEWAY_URL")
	if url == "" {
		url = "http://localhost:8080"
	}
	return url
}

type httpResult struct {
	code int
	body map[string]any
}

func doJSON(t *testing.T, method, url, token string, payload any) httpResult {
	t.Helper()
	var body io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return httpResult{code: resp.StatusCode, body: out}
}

func TestEndToEndFlow(t *testing.T) {
	base := skipIfDisabled(t)
	stamp := time.Now().UnixNano()
	passEmail := fmt.Sprintf("passenger_%d@test.kz", stamp)
	drvEmail := fmt.Sprintf("driver_%d@test.kz", stamp)

	// 1. Health check
	resp, err := http.Get(base + "/health")
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("gateway not healthy: %v", err)
	}
	resp.Body.Close()

	// 2. Register passenger
	r := doJSON(t, "POST", base+"/api/auth/register", "", map[string]any{
		"email": passEmail, "password": "Password123", "full_name": "P",
	})
	if r.code != 201 {
		t.Fatalf("register passenger: %d %v", r.code, r.body)
	}
	passLogin := doJSON(t, "POST", base+"/api/auth/login", "", map[string]any{
		"email": passEmail, "password": "Password123",
	})
	passToken, _ := passLogin.body["access_token"].(string)
	if passToken == "" {
		t.Fatalf("no passenger token: %v", passLogin.body)
	}
	passUser, _ := passLogin.body["user"].(map[string]any)
	passID, _ := passUser["id"].(string)

	// 3. Register driver
	r = doJSON(t, "POST", base+"/api/auth/register", "", map[string]any{
		"email": drvEmail, "password": "Password123", "full_name": "D", "role": "driver",
	})
	if r.code != 201 {
		t.Fatalf("register driver: %d %v", r.code, r.body)
	}
	drvLogin := doJSON(t, "POST", base+"/api/auth/login", "", map[string]any{
		"email": drvEmail, "password": "Password123",
	})
	drvToken, _ := drvLogin.body["access_token"].(string)
	drvUser, _ := drvLogin.body["user"].(map[string]any)
	drvUserID, _ := drvUser["id"].(string)

	// 4. Create driver profile + go online + set location
	r = doJSON(t, "POST", base+"/api/drivers/register", drvToken, map[string]any{
		"user_id": drvUserID, "license_number": "KZ-IT-001",
	})
	if r.code != 201 {
		t.Fatalf("register driver profile: %d %v", r.code, r.body)
	}
	drvBody, _ := r.body["driver"].(map[string]any)
	driverID, _ := drvBody["id"].(string)

	r = doJSON(t, "PATCH", base+"/api/drivers/status", drvToken, map[string]any{
		"driver_id": driverID, "status": "online",
	})
	if r.code != 200 {
		t.Fatalf("driver status: %d %v", r.code, r.body)
	}
	r = doJSON(t, "PATCH", base+"/api/drivers/location", drvToken, map[string]any{
		"driver_id": driverID, "latitude": 51.169392, "longitude": 71.449074,
	})
	if r.code != 200 {
		t.Fatalf("driver location: %d %v", r.code, r.body)
	}

	// 5. Create ride
	r = doJSON(t, "POST", base+"/api/rides", passToken, map[string]any{
		"passenger_id": passID,
		"pickup_lat":   51.169392, "pickup_lng": 71.449074,
		"dropoff_lat": 51.180000, "dropoff_lng": 71.460000,
	})
	if r.code != 201 {
		t.Fatalf("create ride: %d %v", r.code, r.body)
	}
	rideBody, _ := r.body["ride"].(map[string]any)
	rideID, _ := rideBody["id"].(string)
	if rideID == "" {
		t.Fatalf("no ride id: %v", r.body)
	}

	// 6. Wait for NATS-driven driver assignment
	var status string
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		got := doJSON(t, "GET", base+"/api/rides/"+rideID, passToken, nil)
		rb, _ := got.body["ride"].(map[string]any)
		status, _ = rb["status"].(string)
		if status == "driver_assigned" {
			break
		}
	}
	if status != "driver_assigned" {
		t.Fatalf("ride was not auto-assigned (status=%q)", status)
	}

	// 7. Driver finishes the ride
	doJSON(t, "PATCH", base+"/api/rides/"+rideID+"/status", drvToken, map[string]any{"status": "in_progress"})
	r = doJSON(t, "POST", base+"/api/rides/"+rideID+"/complete", drvToken, nil)
	if r.code != 200 {
		t.Fatalf("complete: %d %v", r.code, r.body)
	}

	// 8. Wait for payment auto-creation
	var payStatus string
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		got := doJSON(t, "GET", base+"/api/rides/"+rideID+"/payment", passToken, nil)
		if got.code == 200 {
			pb, _ := got.body["payment"].(map[string]any)
			payStatus, _ = pb["status"].(string)
			if payStatus == "succeeded" {
				break
			}
		}
	}
	if payStatus != "succeeded" {
		t.Fatalf("payment did not succeed (status=%q)", payStatus)
	}
}

package domain

import (
	"errors"
	"math"
	"time"
)

const (
	StatusOnline  = "online"
	StatusOffline = "offline"
	StatusBusy    = "busy"
)

var (
	ErrDriverNotFound = errors.New("driver not found")
	ErrNoDriverNearby = errors.New("no driver available nearby")
	ErrInvalidStatus  = errors.New("invalid driver status")
)

type Driver struct {
	ID            string
	UserID        string
	LicenseNumber string
	Status        string
	Latitude      float64
	Longitude     float64
	CreatedAt     time.Time
}

type Vehicle struct {
	ID          string
	DriverID    string
	PlateNumber string
	Make        string
	Model       string
	Year        int32
	Color       string
}

type NearbyDriver struct {
	DriverID   string
	Latitude   float64
	Longitude  float64
	DistanceKm float64
}

type StatusHistory struct {
	DriverID  string
	Status    string
	ChangedAt time.Time
}

// HaversineKm computes great-circle distance between two coordinates.
func HaversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371.0
	rad := math.Pi / 180.0
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

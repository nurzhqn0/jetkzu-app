package domain

import (
	"errors"
	"math"
	"time"
)

const (
	StatusRequested      = "requested"
	StatusDriverAssigned = "driver_assigned"
	StatusDriverArrived  = "driver_arrived"
	StatusInProgress     = "in_progress"
	StatusCompleted      = "completed"
	StatusCancelled      = "cancelled"
)

var (
	ErrRideNotFound      = errors.New("ride not found")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrRideAlreadyDone   = errors.New("ride already finished")
)

type Ride struct {
	ID          string
	PassengerID string
	DriverID    string
	PickupLat   float64
	PickupLng   float64
	DropoffLat  float64
	DropoffLng  float64
	Status      string
	Price       float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type StatusHistory struct {
	RideID    string
	Status    string
	Reason    string
	ChangedAt time.Time
}

// allowedTransitions enforces a simple FSM for ride status.
var allowedTransitions = map[string]map[string]struct{}{
	StatusRequested:      {StatusDriverAssigned: {}, StatusCancelled: {}},
	StatusDriverAssigned: {StatusDriverArrived: {}, StatusInProgress: {}, StatusCancelled: {}},
	StatusDriverArrived:  {StatusInProgress: {}, StatusCancelled: {}},
	StatusInProgress:     {StatusCompleted: {}, StatusCancelled: {}},
	StatusCompleted:      {},
	StatusCancelled:      {},
}

func CanTransition(from, to string) bool {
	allowed, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	_, ok = allowed[to]
	return ok
}

// EstimatePrice computes a deterministic mock price using haversine distance.
// Base fare 500 KZT + 120 KZT/km. Minimum 700 KZT.
func EstimatePrice(pickupLat, pickupLng, dropoffLat, dropoffLng float64) (price, distanceKm float64) {
	distanceKm = haversineKm(pickupLat, pickupLng, dropoffLat, dropoffLng)
	price = 500 + 120*distanceKm
	if price < 700 {
		price = 700
	}
	return math.Round(price*100) / 100, math.Round(distanceKm*100) / 100
}

func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371.0
	rad := math.Pi / 180.0
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

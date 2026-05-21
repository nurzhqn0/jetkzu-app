package redis

import (
	"context"
	"fmt"

	"github.com/jetkzu/jetkzu/services/driver/internal/domain"
	"github.com/redis/go-redis/v9"
)

const (
	geoKey    = "driver:geo"
	statusKey = "driver:status:"
)

type Cache struct {
	c *redis.Client
}

func New(c *redis.Client) *Cache { return &Cache{c: c} }

func (c *Cache) SetLocation(ctx context.Context, driverID string, lat, lng float64) error {
	return c.c.GeoAdd(ctx, geoKey, &redis.GeoLocation{Name: driverID, Latitude: lat, Longitude: lng}).Err()
}

func (c *Cache) SetStatus(ctx context.Context, driverID, status string) error {
	return c.c.Set(ctx, statusKey+driverID, status, 0).Err()
}

func (c *Cache) GetStatus(ctx context.Context, driverID string) (string, error) {
	v, err := c.c.Get(ctx, statusKey+driverID).Result()
	if err == redis.Nil {
		return domain.StatusOffline, nil
	}
	if err != nil {
		return "", err
	}
	return v, nil
}

func (c *Cache) FindNearest(ctx context.Context, lat, lng, radiusKm float64, limit int) ([]domain.NearbyDriver, error) {
	if limit <= 0 {
		limit = 10
	}
	res, err := c.c.GeoSearchLocation(ctx, geoKey, &redis.GeoSearchLocationQuery{
		GeoSearchQuery: redis.GeoSearchQuery{
			Longitude:  lng,
			Latitude:   lat,
			Radius:     radiusKm,
			RadiusUnit: "km",
			Sort:       "ASC",
			Count:      limit,
		},
		WithCoord: true,
		WithDist:  true,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("geo search: %w", err)
	}
	out := make([]domain.NearbyDriver, 0, len(res))
	for _, r := range res {
		st, _ := c.GetStatus(ctx, r.Name)
		if st != domain.StatusOnline {
			continue
		}
		out = append(out, domain.NearbyDriver{
			DriverID:   r.Name,
			Latitude:   r.Latitude,
			Longitude:  r.Longitude,
			DistanceKm: r.Dist,
		})
	}
	return out, nil
}

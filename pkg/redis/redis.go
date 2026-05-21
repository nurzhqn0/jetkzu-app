package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func Connect(ctx context.Context, addr string) (*redis.Client, error) {
	c := redis.NewClient(&redis.Options{Addr: addr})
	deadline := time.Now().Add(30 * time.Second)
	for {
		if err := c.Ping(ctx).Err(); err == nil {
			return c, nil
		} else if time.Now().After(deadline) {
			return nil, fmt.Errorf("redis not reachable: %w", err)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}

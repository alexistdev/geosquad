// Package cache membungkus Redis. Dipakai untuk menyimpan sesi login dan
// menyiarkan log run ke klien SSE.
package cache

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/alexistdev/geosquad/api/internal/config"
)

type Client struct {
	*redis.Client
}

func Open(cfg config.RedisConfig) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("ping redis %s: %w", cfg.Addr(), err)
	}

	slog.Info("redis tersambung", "addr", cfg.Addr(), "db", cfg.DB)
	return &Client{Client: rdb}, nil
}

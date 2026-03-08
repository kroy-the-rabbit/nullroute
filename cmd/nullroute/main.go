package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kroy-the-rabbit/nullroute/internal/config"
	"github.com/kroy-the-rabbit/nullroute/internal/sources"
	"github.com/kroy-the-rabbit/nullroute/internal/syncer"
)

func main() {
	var cfgPath string
	flag.StringVar(&cfgPath, "config", "/etc/nullroute/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("failed loading config: %v", err)
	}

	engine := syncer.NewEngine(cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if cfg.RunOnce {
		if err := runOnce(ctx, engine); err != nil {
			log.Fatalf("sync failed: %v", err)
		}
		return
	}

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	if err := runOnce(ctx, engine); err != nil {
		log.Printf("initial sync failed: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("shutting down")
			return
		case <-ticker.C:
			if err := runOnce(ctx, engine); err != nil {
				log.Printf("periodic sync failed: %v", err)
			}
		}
	}
}

func runOnce(parent context.Context, engine *syncer.Engine) error {
	// Initial full-table loads can exceed a few minutes on cold start.
	ctx, cancel := context.WithTimeout(parent, 20*time.Minute)
	defer cancel()

	prefixes, err := sources.FetchAndParse(ctx, engine.Config())
	if err != nil {
		return err
	}

	return engine.Sync(ctx, prefixes)
}

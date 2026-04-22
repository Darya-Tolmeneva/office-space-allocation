package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"office-space-allocation/apps/backend/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	application, err := app.New()
	if err != nil {
		log.Fatalf("build application: %v", err)
	}

	if err := application.Run(ctx); err != nil {
		log.Fatalf("run application: %v", err)
	}
}

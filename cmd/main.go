package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/meindokuse/task-service/internal"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := internal.New(ctx).Run(ctx); err != nil {
		log.Fatal(err)
	}
}

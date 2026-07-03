package main

import (
	"context"
	"os"

	"idmagic/internal/bootstrap"
	"idmagic/internal/shared/logging"
)

func main() {
	if err := bootstrap.Run(); err != nil {
		logging.Error(context.Background(), "idmagic exited with error", "error", err)
		os.Exit(1)
	}
}

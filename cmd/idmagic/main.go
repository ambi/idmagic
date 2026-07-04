package main

import (
	"context"
	"os"

	"github.com/ambi/idmagic/internal/bootstrap"
	"github.com/ambi/idmagic/internal/shared/logging"
)

func main() {
	if err := bootstrap.Run(); err != nil {
		logging.Error(context.Background(), "idmagic exited with error", "error", err)
		os.Exit(1)
	}
}

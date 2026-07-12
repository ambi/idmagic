package main

import (
	"context"
	"os"

	"github.com/ambi/idmagic/backend/shared/logging"
)

func main() {
	if err := Run(); err != nil {
		logging.Error(context.Background(), "idmagic exited with error", "error", err)
		os.Exit(1)
	}
}

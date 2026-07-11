package main

import (
	"context"
	"os"

	"github.com/ambi/idmagic/backend/bootstrap"
	"github.com/ambi/idmagic/backend/shared/logging"
)

func main() {
	if err := bootstrap.RunWorker(); err != nil {
		logging.Error(context.Background(), "idmagic worker exited with error", "error", err)
		os.Exit(1)
	}
}

package main

import (
	"context"
	"os"

	"github.com/ambi/idmagic/backend/relay"
	"github.com/ambi/idmagic/backend/shared/logging"
)

func main() {
	if err := relay.Run(); err != nil {
		logging.Error(context.Background(), "idmagic relay exited with error", "error", err)
		os.Exit(1)
	}
}

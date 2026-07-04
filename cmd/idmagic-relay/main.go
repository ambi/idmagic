package main

import (
	"context"
	"os"

	"github.com/ambi/idmagic/internal/relay"
	"github.com/ambi/idmagic/internal/shared/logging"
)

func main() {
	if err := relay.Run(); err != nil {
		logging.Error(context.Background(), "idmagic relay exited with error", "error", err)
		os.Exit(1)
	}
}

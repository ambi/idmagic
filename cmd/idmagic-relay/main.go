package main

import (
	"log"

	"idmagic/internal/relay"
)

func main() {
	if err := relay.Run(); err != nil {
		log.Fatal(err)
	}
}

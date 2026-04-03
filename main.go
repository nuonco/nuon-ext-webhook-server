package main

import (
	"os"

	"github.com/nuonco/nuon-ext-webhook-server/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

package main

import (
	"os"

	"github.com/aereal/enimore/internal/cli"
)

func main() {
	os.Exit(cli.New().Run(os.Args))
}

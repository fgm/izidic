package main

import (
	"log"
	"os"

	"github.com/fgm/izidic/examples/di"
)

func main() {
	dic := di.Resolve(os.Stdout, os.Args[0], os.Args[1:])
	app := dic.MustService("app").(di.App)
	log.Printf("app: %#v\n", app)
	if err := app(); err != nil {
		os.Exit(1)
	}
}

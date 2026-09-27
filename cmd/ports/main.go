package main

import (
	"ports/internal/cli"
)

// version can be injected at build time using -ldflags "-X main.version=x.y.z"
var version = "0.1.2"

func main() {
	cli.Execute(version)
}

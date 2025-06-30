package main

import (
	"adapter/cmd"
	"adapter/module/fiber"
)

func main() {
	cmd.Initialized()
	fiber.Initialized(":3000")
}

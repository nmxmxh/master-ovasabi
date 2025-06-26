package main

import (
	"github.com/nmxmxh/master-ovasabi/internal/server"
)

func main() {
	// Delegate server startup and lifecycle management to server.Run
	server.Run()
}

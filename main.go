package main

import (
	"github.com/meateam/permit-service/server"
)

func main() {
	server.NewServer(nil).Serve(nil)
}

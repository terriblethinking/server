package main

import (
	// "log"
	// "net/http"

	"github.com/terriblethinking/core/gateway"
)

func main() {

	client, err := gateway.InitGateway()

	if err != nil {
		panic(err)
	}

	defer client.Shutdown()




}


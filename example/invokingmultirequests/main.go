package main

// This example shows a basic usage
// For more advanced usages, please refer more examples.

import (
	"context"
	"github.com/hyuti/minion"
	"log"
	"time"
)

type response struct {
	msg string
}

func invokeRequest(_ context.Context) *response {
	//	not doing special
	//	just simulate a long-running invoking request
	log.Println("sleeping in 2 seconds")
	time.Sleep(2 * time.Second)
	return &response{
		msg: "done",
	}
}
func main() {
	gru := minion.New[*response]()
	defer gru.Clean()

	// WithEvent allows us to add necessary logics after every complete minion
	gru.WithEvent(func(r *response) {
		if r == nil {
			return
		}
		log.Println(r.msg)
	})
	// don't forget to call start after setting up minions
	gru.Start(func() *response {
		return invokeRequest(context.Background())
	}, func() *response {
		return invokeRequest(context.Background())
	})

	// handle error happened among the minions
	if err := gru.Error(); err != nil {
		log.Println(err)
	}
}

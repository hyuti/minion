package main

// This example demonstrates timeout constraint set up for minions
// to utilize timeout feature we use context with timeout provided by gopkg

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
	log.Println("sleeping in 2 seconds")
	// took longer than 1 second which is timeout
	time.Sleep(2 * time.Second)
	return &response{
		msg: "done",
	}
}
func main() {
	gru := minion.New[*response]()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	// to release resource
	defer cancel()

	// use AddMinionWithCtx instead of AddMinion to pass context with timeout already defined above.
	gru.AddMinionWithCtx(ctx, func(ctx context.Context) *response {
		return invokeRequest(ctx)
	})
	gru.AddMinionWithCtx(ctx, func(ctx context.Context) *response {
		return invokeRequest(ctx)
	})

	// the rest is the same as the invokingmultirequests example
	gru.WithEvent(func(r *response) {
		if r == nil {
			return
		}
		log.Println(r.msg)
	})
	gru.Start()

	// error at this moment must not be nil because of timeout elapsed
	// the output will be like:
	//[time] sleeping in 2 seconds
	//[time] sleeping in 2 seconds
	//[time] timeout elapsed: context deadline exceeded
	//       timeout elapsed: context deadline exceeded
	if err := gru.Error(); err != nil {
		log.Println(err)
	}
}

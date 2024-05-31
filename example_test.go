package minion_test

import (
	"context"
	"fmt"
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
	fmt.Println("sleeping in 2 seconds")
	// Output: sleeping in 2 seconds
	time.Sleep(2 * time.Second)
	return &response{
		msg: "done",
	}
}

func ExampleGru_Start() {
	gru := minion.New[*response]()
	defer gru.Clean()

	// WithEvent allows us to add necessary logics after every complete minion
	gru.WithEvent(func(r *response) {
		if r == nil {
			return
		}
		fmt.Println(r.msg)
		// Output: done
	})

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

func ExampleGru_StartWithCtx() {
	gru := minion.New[*response]()
	defer gru.Clean()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	// to release resource
	defer cancel()

	gru.WithEvent(func(r *response) {
		if r == nil {
			return
		}
		fmt.Println(r.msg)
		// Output: done
	})

	// use StartWithCtx instead of Start to pass context with timeout already defined above.
	gru.StartWithCtx(ctx, func(ctx context.Context) *response {
		return invokeRequest(ctx)
	}, func(ctx context.Context) *response {
		return invokeRequest(ctx)
	})

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

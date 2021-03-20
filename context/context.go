package main

import (
	"context"
	"fmt"
	"time"
)

func infiniteLoop(ctx context.Context) {
	innerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	for {
		fmt.Println("waiting for time out")
		select {
		case <-innerCtx.Done():
			fmt.Println("Exit now")
			fmt.Println("message:", ctx.Value("message").(string))
			return
		default:
		}
	}
}

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	ctx = context.WithValue(ctx, "message", "hi")
	defer cancel()
	go infiniteLoop(ctx)
	select {
	case <-ctx.Done():
		fmt.Println(ctx.Err())
	}
}

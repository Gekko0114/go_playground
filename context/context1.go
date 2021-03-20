package main

import (
	"context"
	"fmt"
)

func main() {
	//https://twinbird-htn.hatenablog.com/entry/2017/04/07/214420
	ctx := context.Background()
	ctx = context.WithValue(ctx, "hoge", 1)
	fmt.Println(ctx.Value("hoge").(int))
}

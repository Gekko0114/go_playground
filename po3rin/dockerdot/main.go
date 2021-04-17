package main

import (
	"dockerdot/docker2dot"
	"fmt"
	"syscall/js"
)

func registerCallbacks() {
	var cb js.Func
	document := js.Global().Get("document")
	element := document.Call("getElementById", "textarea")

	cb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		text := element.Get("value").String()
		dockerfile := []byte(text)

		go func() {
			dot, err := docker2dot.Docker2Dot(dockerfile)
			if err != nil {
				fmt.Println(err)
			}
			showGraph := js.Global().Get("showGraph")
			showGraph.Invoke(string(dot))
		}()
		return nil
	})
	js.Global().Get("document").Call("getElementById", "button").Call("addEventListener", "click", cb)
}

func main() {
	c := make(chan struct{}, 0)
	registerCallbacks()
	<-c
}

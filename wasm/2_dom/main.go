package main

import "syscall/js"

func main() {
	window := js.Global()
	message := window.Get("document").Call("getElementById", "message")

	message.Set("innerHTML", "hello WebAssembly")
}

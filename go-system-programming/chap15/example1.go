package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Printf("page size: %d\n", os.Getpagesize())
}

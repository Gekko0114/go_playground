package main

import (
	"fmt"
	"time"
)

func main() {
	var sec int
	fmt.Println("please type the number")
	fmt.Scan(&sec)
	fmt.Println("start", time.Now())
	<-time.After(time.Duration(sec) * time.Second)
	fmt.Println("end", time.Now())
}

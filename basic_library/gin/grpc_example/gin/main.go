package main

import (
	"fmt"
	"log"
	"net/http"

	pb "gin/pb"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewGreeterClient(conn)

	r := gin.Default()
	r.GET("/rest/n/:name", func(c *gin.Context) {
		name := c.Param("name")

		req := &pb.HelloRequest{Name: name}
		res, err := client.SayHello(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"result": fmt.Sprint(res.Message),
		})
	})

	if err := r.Run(":8052"); err != nil {
		log.Fatalf("could not run server: %v", err)
	}
}

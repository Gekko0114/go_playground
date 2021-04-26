package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	newrelic "github.com/newrelic/go-agent"
)

const (
	NewRelicTxnKey = "NewRelicTxnKey"
)

func NewRelicMonitoring(app newrelic.Application) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		txn := app.StartTransaction(ctx.Request.URL.Path, ctx.Writer, ctx.Request)
		defer txn.End()
		ctx.Set(NewRelicTxnKey, txn)
		ctx.Next()
	}
}

func main() {
	router := gin.Default()

	cfg := newrelic.NewConfig(os.Getenv("APP_NAME"), os.Getenv("NEW_RELIC_API_KEY"))
	app, err := newrelic.NewApplication(cfg)
	if err != nil {
		log.Printf("failed to make new_relic app: %v", err)
	} else {
		router.Use(NewRelicMonitoring(app))
	}

	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello World\n")
	})
	router.Run()
}

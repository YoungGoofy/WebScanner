package app

import (
	"log"

	"github.com/YoungGoofy/WebScanner/internal/transport"
	"github.com/gin-gonic/gin"
)

func Run() {
	router := gin.Default()
	router.LoadHTMLGlob("frontend/*")
	transport.MainHandler(router)
	log.Println("Listening server on: http://localhost:3000")
	if err := router.Run(":3000"); err != nil {
		log.Println(err)
	}
}

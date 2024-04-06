package app

import (
	"github.com/YoungGoofy/WebScanner/internal/transport"
	"github.com/gin-gonic/gin"
	"log"
)

func Run() {
	router := gin.Default()
	//router.Use(static.Serve("/", static.LocalFile("./web/views", true)))
	router.LoadHTMLGlob("web/views/*")
	transport.MainHandler(router)
	if err := router.Run(":3000"); err != nil {
		log.Println(err)
	}
	log.Println("Listening server on: http://localhost:3000")
}

package transport

import (
	"github.com/YoungGoofy/WebScanner/internal/services/alerts"
	"github.com/YoungGoofy/WebScanner/internal/services/scan"
	"github.com/YoungGoofy/WebScanner/internal/utils/toml"
	"github.com/gin-gonic/gin"
	"net/http"
)

func MainHandler(api *gin.Engine) {
	key := toml.GetApiKeyFromToml()
	s := scan.NewScanner(key)
	a := alerts.NewAlerts(*s)
	api.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "Home Page",
		})
	})
	api.GET("/scan", s.StartScan)
	api.GET("/alerts", a.GetAlerts)
	// api.GET("/scan/alerts/:cwe_id/:page", a.GetTotalCommonAlerts)
	// api.GET("/alert/:id", a.GetOnlyAlert)
	api.POST("/addKey", postAPIKey)
	//api.POST("/stop", s.StopScan)
	//api.POST("/pause", s.PauseScan)
	//api.POST("/resume", s.ResumeScan)
	api.POST("/action/:action", s.HandleAction)
}

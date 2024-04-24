package transport

import (
	"github.com/YoungGoofy/WebScanner/internal/transport/alerts"
	"github.com/YoungGoofy/WebScanner/internal/transport/scan"
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
	api.GET("/scan/alerts", a.GetAlerts)
	api.GET("/scan/alerts/:cwe_id/:page", a.GetTotalCommonAlerts)
	api.GET("/alert/:id", a.GetOnlyAlert)
	api.GET("/settings", func(c *gin.Context) {
		c.HTML(http.StatusOK, "settings.html", gin.H{
			"title": "Settings Page",
			"key":   key,
		})
	})
	api.POST("/settings/addKey", postAPIKey)
	//api.POST("/stop", s.StopScan)
	//api.POST("/pause", s.PauseScan)
	//api.POST("/resume", s.ResumeScan)
	api.POST("/action/:action", s.HandleAction)
}

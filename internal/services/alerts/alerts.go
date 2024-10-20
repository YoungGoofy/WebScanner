package alerts

import (
	"github.com/YoungGoofy/WebScanner/internal/services/scan"
	"github.com/YoungGoofy/gozap/pkg/gozap"
	"github.com/YoungGoofy/gozap/pkg/models"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

type (
	Alerts struct {
		s                   *scan.Scanner
		groupOfCommonAlerts groupOfCommonAlerts
	}
	CommonAlert struct {
		CweId             string
		Count             int
		Name              string
		TotalCommonAlerts []models.Alert
	}
	groupOfCommonAlerts struct {
		CommonAlerts       []CommonAlert
		actualListOfAlerts []models.Alert
	}
)

func NewAlerts(scanner scan.Scanner) *Alerts {
	return &Alerts{s: &scanner}
}

func (a *Alerts) GetAlerts(c *gin.Context) {
	main := a.s.MainScanner
	countOfAlerts, err := main.CountOfAlerts()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
	}
	a.groupOfCommonAlerts = groupOfCommonAlerts{make([]CommonAlert, 0, 32), make([]models.Alert, 0, 32)}
	err = a.groupOfCommonAlerts.commonAlerts(countOfAlerts, main)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
	}
	c.HTML(http.StatusOK, "headerAlerts.html", gin.H{
		"title":  "Alerts",
		"count":  countOfAlerts,
		"alerts": a.groupOfCommonAlerts.CommonAlerts,
	})
}

func (g *groupOfCommonAlerts) commonAlerts(countOfAlerts string, main gozap.MainScan) error {

	listOfAlerts, err := main.GetAlerts("0", countOfAlerts)
	if err != nil {
		return err
	}

	for _, item := range listOfAlerts.Alert {
		found := false
		for i, listItem := range g.CommonAlerts {
			if listItem.CweId == item.CweId {
				g.CommonAlerts[i].TotalCommonAlerts = append(g.CommonAlerts[i].TotalCommonAlerts, item)
				g.CommonAlerts[i].Count++
				found = true
				break
			}
		}
		if !found {
			var totalCommonAlerts = make([]models.Alert, 0, 256)
			totalCommonAlerts = append(totalCommonAlerts, item)
			tempItem := CommonAlert{CweId: item.CweId, Name: item.Alert, Count: 1, TotalCommonAlerts: totalCommonAlerts}
			g.CommonAlerts = append(g.CommonAlerts, tempItem)
		}
	}
	return nil
}

type Pagination struct {
	PrevPage int
	NextPage int
	CurrPage int
}

func (a *Alerts) GetTotalCommonAlerts(c *gin.Context) {
	cweId := c.Param("cwe_id")
	page, err := strconv.Atoi(c.Param("page"))
	startIndex := (page - 1) * 25
	endIndex := page * 25
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
	}

	values := a.groupOfCommonAlerts.getAlertsFromCweId(cweId)
	if values == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No cwe_id",
		})
	}
	if endIndex > len(values) {
		endIndex = len(values)
	}
	c.HTML(http.StatusOK, "totalAlerts.html", gin.H{
		"cwe_id": cweId,
		"values": values[startIndex:endIndex],
		"pagination": Pagination{
			PrevPage: page - 1,
			CurrPage: page,
			NextPage: page + 1,
		},
	})
}

func (g *groupOfCommonAlerts) getAlertsFromCweId(cweId string) []models.Alert {
	for _, item := range g.CommonAlerts {
		if cweId == item.CweId {
			g.actualListOfAlerts = item.TotalCommonAlerts
			return item.TotalCommonAlerts
		}
	}
	return nil
}

func (a *Alerts) GetOnlyAlert(c *gin.Context) {
	id := c.Param("id")
	errorAlert := models.Alert{ID: "-1"}
	value := a.groupOfCommonAlerts.getAlertFromId(id)
	if value.ID == errorAlert.ID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No alert with this id",
		})
	}
	c.HTML(http.StatusOK, "alert.html", gin.H{
		"title": value.Alert,
		"value": value,
	})
}

func (g *groupOfCommonAlerts) getAlertFromId(id string) models.Alert {
	for _, alert := range g.actualListOfAlerts {
		if alert.ID == id {
			return alert
		}
	}
	return models.Alert{ID: "-1"}
}

package alerts

import (
	"github.com/YoungGoofy/WebScanner/internal/transport/scan"
	"github.com/YoungGoofy/gozap/pkg/gozap"
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
		TotalCommonAlerts []struct {
			Id          string
			Method      string
			Url         string
			Description string
		}
	}
	groupOfCommonAlerts struct {
		CommonAlerts []CommonAlert
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
	a.groupOfCommonAlerts = groupOfCommonAlerts{make([]CommonAlert, 0, 32)}
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
				g.CommonAlerts[i].TotalCommonAlerts = append(g.CommonAlerts[i].TotalCommonAlerts, struct {
					Id          string
					Method      string
					Url         string
					Description string
				}{Id: item.ID, Method: item.Method, Url: item.URL, Description: item.Description})
				g.CommonAlerts[i].Count++
				found = true
				break
			}
		}
		if !found {
			var totalCommonAlerts = make([]struct {
				Id          string
				Method      string
				Url         string
				Description string
			}, 0, 256)
			totalCommonAlerts = append(totalCommonAlerts, struct {
				Id          string
				Method      string
				Url         string
				Description string
			}{Id: item.ID, Method: item.Method, Url: item.URL, Description: item.Description})
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

	values := a.groupOfCommonAlerts.getAlertsFromId(cweId)
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

func (g *groupOfCommonAlerts) getAlertsFromId(cweId string) []struct {
	Id          string
	Method      string
	Url         string
	Description string
} {
	for _, item := range g.CommonAlerts {
		if cweId == item.CweId {
			return item.TotalCommonAlerts
		}
	}
	return nil
}

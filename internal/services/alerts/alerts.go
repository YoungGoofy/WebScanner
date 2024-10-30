package alerts

import (
	"net/http"
	"strconv"

	"github.com/YoungGoofy/WebScanner/internal/services/scan"
	"github.com/YoungGoofy/gozap/pkg/gozap"
	"github.com/YoungGoofy/gozap/pkg/models"
	"github.com/gin-gonic/gin"
)

type (
	Alerts struct {
		s     *scan.Scanner
		risks map[string][]CommonAlert
	}
	groupOfCommonAlerts struct {
		CommonAlerts       []CommonAlert
		actualListOfAlerts []models.Alert
	}
	CommonAlert struct {
		CweId             string
		Count             int
		Name              string
		Risk              string
		TotalCommonAlerts []models.Alert
	}
)

func NewAlerts(scanner scan.Scanner) *Alerts {
	r := make(map[string][]CommonAlert)
	return &Alerts{s: &scanner, risks: r}
}

func (a *Alerts) GetAlerts(c *gin.Context) {
	main := a.s.MainScanner
	countOfAlerts, err := main.CountOfAlerts()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
	}

	a.risks, err = commonRisks(countOfAlerts, main)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"title": "Alerts",
		"risks": a.risks,
	})
}

func commonRisks(countOfAlerts string, main gozap.MainScan) (map[string][]CommonAlert, error) {
	listOfAlerts, err := main.GetAlerts("0", countOfAlerts)
	if err != nil {
		return nil, err
	}
	riskMap := map[string][]CommonAlert{
		"Informational": {},
		"Low":           {},
		"Medium":        {},
		"High":          {},
	}

	// Создаем мапу для хранения уникальных CweId с массивом всех TotalCommonAlerts
	alertMap := make(map[string]*CommonAlert)

	for _, item := range listOfAlerts.Alert {
		// riskLevel := item.Risk // определяем уровень риска для текущего алерта
		// Проверяем, существует ли уже алерт с этим CweId
		if existingAlert, exists := alertMap[item.CweId]; exists {
			existingAlert.TotalCommonAlerts = append(existingAlert.TotalCommonAlerts, item)
			existingAlert.Count++
		} else {
			// Создаем новый алерт и добавляем его в список соответствующего уровня риска
			totalCommonAlerts := make([]models.Alert, 0, 256)
			totalCommonAlerts = append(totalCommonAlerts, item)
			newAlert := CommonAlert{
				CweId:             item.CweId,
				Name:              item.Alert,
				Count:             1,
				Risk:              item.Risk,
				TotalCommonAlerts: totalCommonAlerts,
			}

			// Добавляем новый алерт в alertMap и соответствующий уровень риска
			alertMap[item.CweId] = &newAlert
			// riskMap[riskLevel] = append(riskMap[riskLevel], newAlert)
		}
	}

	for _, alert := range alertMap {
		riskMap[alert.Risk] = append(riskMap[alert.Risk], *alert)
	}

	return riskMap, nil
}





type Pagination struct {
	PrevPage int
	NextPage int
	CurrPage int
}

func GetTotalCommonAlerts(c *gin.Context) {
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

func GetOnlyAlert(c *gin.Context) {
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

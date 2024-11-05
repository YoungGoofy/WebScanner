package alerts

import (
	"net/http"
	"sync"
	"time"

	"github.com/YoungGoofy/WebScanner/internal/services/scan"
	"github.com/YoungGoofy/gozap/pkg/gozap"
	"github.com/YoungGoofy/gozap/pkg/models"
	"github.com/gin-gonic/gin"
)

type (
	Alerts struct {
		scanner *scan.Scanner
		risks   map[string][]CommonAlert
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
	return &Alerts{scanner: &scanner, risks: r}
}

func (a *Alerts) GetAlerts(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	
	main := a.scanner.MainScanner
	ascan := a.scanner.ActiveScanner

	var wg sync.WaitGroup
	lastAlertCh := make(chan models.Alert)
	errCh := make(chan error)
	statusCh := make(chan string)
	done := make(chan struct{})
	ticker := time.Tick(250 * time.Millisecond)

	wg.Add(1)
	go func() {
		defer wg.Done()
		commonRisks(main, ascan, lastAlertCh, errCh, statusCh, done)
	}()

	for range ticker {
		select {
		case alert := <-lastAlertCh:
			c.SSEvent("alerts", map[string]any{
				"name":       alert.Name,
				"risk":       alert.Risk,
				"method":     alert.Method,
				"url":        alert.URL,
				"cweid":      alert.CweId,
				"desciption": alert.Description,
				"solution":   alert.Solution,
				"attack":     alert.Attack,
			})
			c.Writer.Flush()
		case err := <-errCh:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
		case status := <-statusCh:
			if status == "100" {
				close(done)
			}
		case <-done:
			wg.Wait()
			close(lastAlertCh)
			close(errCh)
			close(statusCh)
			return
		}
	}

	// c.JSON(http.StatusOK, gin.H{
	// 	"title": "Alerts",
	// 	"risks": a.risks,
	// })
}

func commonRisks(main gozap.MainScan,
	ascan gozap.ActiveScanner,
	lastAlertCh chan<- models.Alert,
	errCh chan<- error,
	statusCh chan string,
	done <-chan struct{}) {

	minCount := "0"

	// Создаем мапу для хранения уникальных CweId с массивом всех TotalCommonAlerts
	// alertMap := make(map[string]*CommonAlert)

	for {
		select {
		case <-done:
			return
		default:
			maxCount, _ := main.CountOfAlerts()
			listOfAlerts, err := main.GetAlerts(minCount, maxCount)
			if err != nil {
				errCh <- err
			}
			if len(listOfAlerts.Alert) > 0 {
				// Проверяем, существует ли уже алерт с этим CweId
				for _, item := range listOfAlerts.Alert {
					lastAlertCh <- item
					// if existingAlert, exists := alertMap[item.CweId]; exists {
					// 	existingAlert.TotalCommonAlerts = append(existingAlert.TotalCommonAlerts, item)
					// 	existingAlert.Count++
					// 	lastAlertCh <- *existingAlert
					// } else {
					// 	// Создаем новый алерт и добавляем его в список соответствующего уровня риска
					// 	totalCommonAlerts := make([]models.Alert, 0, 512)
					// 	totalCommonAlerts = append(totalCommonAlerts, item)
					// 	newAlert := CommonAlert{
					// 		CweId:             item.CweId,
					// 		Name:              item.Alert,
					// 		Count:             1,
					// 		Risk:              item.Risk,
					// 		TotalCommonAlerts: totalCommonAlerts,
					// 	}
					// 	// Добавляем новый алерт в alertMap и соответствующий уровень риска
					// 	alertMap[item.CweId] = &newAlert
					// 	lastAlertCh <- newAlert
					// }
				}
				minCount = maxCount
			}
		}
		status, err := ascan.GetStatus()
		if err != nil {
			errCh <- err
		}
		statusCh <- status
	}
}

/* type Pagination struct {
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
} */

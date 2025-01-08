package alerts

import (
	"context"
	"sync"
	"time"

	"github.com/YoungGoofy/WebScanner/internal/services/scan"
	"github.com/YoungGoofy/gozap/pkg/gozap"
	"github.com/YoungGoofy/gozap/pkg/models"
	"github.com/gin-gonic/gin"
)

// Alerts хранит сканер и карту рисков (для примера).
type Alerts struct {
	scanner *scan.Scanner
	risks   map[string][]CommonAlert
}

// CommonAlert описывает общую структуру для алертов.
type CommonAlert struct {
	CweId             string
	Count             int
	Name              string
	Risk              string
	TotalCommonAlerts []models.Alert
}

// NewAlerts создаёт новый объект Alerts с переданным сканером.
func NewAlerts(scanner scan.Scanner) *Alerts {
	r := make(map[string][]CommonAlert)
	return &Alerts{
		scanner: &scanner,
		risks:   r,
	}
}

// GetAlerts обрабатывает SSE-запрос. Отправляет события об алертах.
// Завершается, когда скан активного анализа возвращает статус "100" или по отмене контекста.
func (a *Alerts) GetAlerts(c *gin.Context) {
	// Устанавливаем заголовки для SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// Создаём контекст, который отменится при разрыве соединения клиентом или при timeout
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	mainScan := a.scanner.MainScan
	activeScan := a.scanner.ActiveScan

	// Создаём каналы для обмена данными
	lastAlertCh := make(chan models.Alert, 100) // буфер для избежания блокировок
	errCh := make(chan error, 10)
	statusCh := make(chan string, 10)

	// Запускаем горутину, которая будет собирать алерты
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		a.collectAlerts(ctx, mainScan, activeScan, lastAlertCh, errCh, statusCh)
	}()

	// Основной цикл чтения из каналов
	//
	// Если хотим регулярно «пинговать» клиента пустым событием,
	// чтобы соединение не рвалось, можем в select добавить:
	// case <-time.After(250 * time.Millisecond):
	// 	// отправляем, например, ping-событие.
	for {
		select {
		case <-ctx.Done():
			// Клиент закрыл соединение или отмена по таймауту — выходим
			close(lastAlertCh)
			close(errCh)
			close(statusCh)
			wg.Wait()
			return

		case alert := <-lastAlertCh:
			c.SSEvent("alerts", map[string]any{
				"id":         alert.ID,
				"name":       alert.Name,
				"risk":       alert.Risk,
				"method":     alert.Method,
				"url":        alert.URL,
				"cweid":      alert.CweId,
				"description": alert.Description, // исправил "desciption" -> "description"
				"solution":   alert.Solution,
			})
			// Важно «проталкивать» ответ в реальном времени
			c.Writer.Flush()

		case err := <-errCh:
			// В зависимости от задачи: можно завершать SSE при ошибке,
			// либо отправлять ошибку и продолжать.
			c.SSEvent("error", gin.H{"error": err.Error()})
			c.Writer.Flush()

		case status := <-statusCh:
			if status == "100" {
				// Скан завершается — отменяем контекст
				cancel()
			}
		}
	}
}

// collectAlerts запускается в отдельной горутине и регулярно собирает алерты из MainScan,
// а также отслеживает статус ActiveScan.
func (a *Alerts) collectAlerts(
	ctx context.Context,
	main gozap.MainScan,
	ascan gozap.ActiveScanner,
	lastAlertCh chan<- models.Alert,
	errCh chan<- error,
	statusCh chan<- string,
) {
	minCount := "0"

	for {
		// Перед каждой итерацией проверяем, не отменён ли контекст
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Получаем текущее количество алертов
		maxCount, err := main.CountOfAlerts()
		if err != nil {
			select {
			case errCh <- err:
			case <-ctx.Done():
				return
			}
		}

		// Если есть новые алерты, собираем их
		listOfAlerts, err := main.GetAlerts(minCount, maxCount)
		if err != nil {
			select {
			case errCh <- err:
			case <-ctx.Done():
				return
			}
		}

		if len(listOfAlerts.Alert) > 0 {
			for _, item := range listOfAlerts.Alert {
				// Отправляем каждый алерт в канал
				select {
				case lastAlertCh <- item:
				case <-ctx.Done():
					return
				}
			}
			minCount = maxCount
		}

		// Проверяем статус активного сканирования
		status, err := ascan.GetStatus()
		if err != nil {
			select {
			case errCh <- err:
			case <-ctx.Done():
				return
			}
		}

		select {
		case statusCh <- status:
		case <-ctx.Done():
			return
		}

		// При желании — пауза между циклами.
		// Убирая sleep, будем крутиться в «while(true)» максимально быстро.
		// Но обычно стоит дать сканеру время для обновления данных.
		time.Sleep(250 * time.Millisecond)
	}
}

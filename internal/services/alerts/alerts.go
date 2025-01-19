package alerts

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
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
	ratingCh := make(chan float64, 10)

	// Запускаем горутину, которая будет собирать алерты
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		a.collectAlerts(ctx, mainScan, activeScan, lastAlertCh, errCh, statusCh, ratingCh)
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
			// 1. Сначала дожидаемся, пока collectAlerts завершится
			wg.Wait()

			// 2. Теперь можем безопасно закрывать каналы
			close(lastAlertCh)
			close(errCh)
			close(statusCh)
			close(ratingCh)
			return

		case alert := <-lastAlertCh:
			c.SSEvent("alerts", map[string]any{
				"id":          alert.ID,
				"name":        alert.Name,
				"risk":        alert.Risk,
				"method":      alert.Method,
				"url":         alert.URL,
				"cweid":       alert.CweId,
				"description": alert.Description, // исправил "desciption" -> "description"
				"solution":    alert.Solution,
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
		case rating, ok := <-ratingCh:
			if !ok {
				// Канал закрыт
				return
			}
			// NEW LOGIC HERE: отправляем рейтинг в JS как отдельное событие
			c.SSEvent("security_rating", gin.H{"rating": rating})
			c.Writer.Flush()

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
	ratingCh chan<- float64,
) {
	minCount := "0"

	securityRating := make(map[string]map[string]int)

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
			// Пробегаемся по новым алертам
			for _, item := range listOfAlerts.Alert {
				// 1. Отправляем каждый алерт в канал, чтобы сразу вывести в SSE «alerts»
				select {
				case lastAlertCh <- item:
				case <-ctx.Done():
					return
				}

				// 2. Добавляем в нашу карту (Risk => map[VulnName]Count)
				riskMap, exists := securityRating[item.Risk]
				if !exists {
					riskMap = make(map[string]int)
					securityRating[item.Risk] = riskMap
				}
				riskMap[item.Name]++
			}
			minCount = maxCount

			// NEW LOGIC HERE: после каждого «пакета» новых алертов — считаем рейтинг.
			rating := SecurityRisk(securityRating)

			// Посылаем рейтинг в канал, чтобы в GetAlerts отправить SSE
			select {
			case ratingCh <- rating:
			case <-ctx.Done():
				return
			}
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

func SecurityRisk(securityRating map[string]map[string]int) float64 {
	// Весы риска
	weights := map[string]float64{
		"High":          5.0,
		"Medium":        3.0,
		"Low":           2.0,
		"Informational": 1.0,
	}

	// Обёртка для итоговой формулы:
	//  RiskFinal = alpha * (sum / totalCount) * (log10(totalCount + 1) + 1)

	var totalCount int64     // Общее кол-во уязвимостей
	var sum float64          // Сумма взвешенных уязвимостей
	partialSumCh := make(chan float64)

	var wg sync.WaitGroup

	for risk, riskMap := range securityRating {
		riskValue := weights[risk] // Если есть нестандартные ключи - вернётся 0

		wg.Add(1)
		go func(rv float64, subMap map[string]int) {
			defer wg.Done()

			var localSum float64
			for _, count := range subMap {
				atomic.AddInt64(&totalCount, int64(count))
				// Индивидуальный риск = weight * (1 + k * count)
				localSum += rv * float64(count)
			}
			partialSumCh <- localSum
		}(riskValue, riskMap)
	}

	go func() {
		wg.Wait()
		close(partialSumCh)
	}()

	for partial := range partialSumCh {
		sum += partial
	}

	tc := float64(totalCount)
	if tc == 0 {
		return 0
	}

	// Новая формула:
	//   RiskFinal = alpha * (sum / tc) * (log10(tc + 1) + 1)
	riskFinal := (sum/tc) * (math.Log10(tc+1) + 1.0)
	return riskFinal
}

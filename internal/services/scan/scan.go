package scan

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/YoungGoofy/gozap/pkg/gozap"
	"github.com/gin-gonic/gin"
)

// Scanner агрегирует все типы сканеров в одном месте.
type Scanner struct {
	MainScan    gozap.MainScan
	PassiveScan gozap.Spider
	ActiveScan  gozap.ActiveScanner
}

// NewScanner создаёт новую структуру Scanner с необходимыми сканерами и API-ключом.
func NewScanner(apiKey string) *Scanner {
	mainScan := gozap.NewMainScan()
	mainScan.AddApiKey(apiKey)

	activeScan := gozap.NewActiveScanner(*mainScan)
	passiveScan := gozap.NewSpider(*mainScan)

	return &Scanner{
		MainScan:    *mainScan,
		PassiveScan: *passiveScan,
		ActiveScan:  *activeScan,
	}
}

// StartScan инициирует процесс пассивного сканирования, а затем — активного.
// Результаты пассивного сканирования отправляются в виде SSE (Server-Sent Events).
func (s *Scanner) StartScan(c *gin.Context) {
	// Структура для получения URL из запроса
	var scanRequest struct {
		URL string `form:"url"`
	}

	if err := c.ShouldBind(&scanRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Добавляем полученный URL в главный сканер
	s.MainScan.AddUrl(scanRequest.URL)

	// Инициализируем объекты сканеров для текущего запроса (нельзя переиспользовать один и тот же MainScan для разных запросов?)
	// Но если необходимо, можно оставить s.PassiveScan и s.ActiveScan
	pScan := gozap.NewSpider(s.MainScan)
	aScan := gozap.NewActiveScanner(s.MainScan)

	// Запускаем пассивное сканирование
	if err := pScan.StartPassiveScan(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Подготавливаем заголовки для SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// Создаём контекст, который завершится при закрытии соединения клиентом
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// При разрыве соединения клиентом поток SSE автоматически завершится.
	// Можно проверить ctx.Done() внутри горутин, чтобы завершить их корректно.

	// Отправляем начальное состояние сканирования (0%)
	sendSSEProgress(c, "0", false)

	// Горутина для обработки результатов пассивного сканирования
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		handlePassiveScanResults(ctx, c, pScan, aScan)
	}()

	// Дожидаемся окончания пассивного сканирования
	wg.Wait()

	// По завершении сообщаем о 100% прогресса (если логика подразумевает именно тут «конец»)
	sendSSEProgress(c, "100", true)
}

// HandleAction управляет командами управления активным сканированием (остановка, пауза, возобновление).
func (s *Scanner) HandleAction(c *gin.Context) {
	action := c.Param("action")

	var err error
	switch action {
	case "stop":
		err = s.ActiveScan.StopScan()
	case "pause":
		err = s.ActiveScan.PauseScan()
	case "resume":
		err = s.ActiveScan.ResumeScan()
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action"})
		return
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

// sendSSEProgress отправляет событие SSE с прогрессом и информацией о завершении сканирования.
func sendSSEProgress(c *gin.Context, progress string, completed bool) {
	sendSSEEvent(c, "progress", map[string]interface{}{
		"progressPercentage": progress,
		"completed":          completed,
	})
}

// sendSSEEvent — утилита для отправки SSE-события и моментального flush.
func sendSSEEvent(c *gin.Context, eventName string, data interface{}) {
	c.SSEvent(eventName, data)
	c.Writer.Flush()
}

// handlePassiveScanResults обрабатывает результаты пассивного сканирования в режиме реального времени
// и при достижении статуса "100" может запускать активное сканирование.
func handlePassiveScanResults(
	ctx context.Context,
	c *gin.Context,
	pScan *gozap.Spider,
	aScan *gozap.ActiveScanner,
) {
	dataChan := make(chan gozap.UrlsInScope)
	errChan := make(chan error)
	progressChan := make(chan string)

	// Можно вынести doneChan, но если используем context, он в целом не обязателен
	doneChan := make(chan struct{})

	// Запускаем чтение результатов пассивного сканирования
	go pScan.AsyncGetResult(dataChan, errChan, progressChan, doneChan)

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Клиент отключился или другой сценарий отмены — завершаемся
			return

		case urls := <-dataChan:
			for _, url := range urls {
				sendSSEEvent(c, "results", map[string]string{
					"processed":          url.Processed,
					"statusReason":       url.StatusReason,
					"method":             url.Method,
					"reasonNotProcessed": url.ReasonNotProcessed,
					"messageId":          url.MessageID,
					"url":                url.URL,
					"statusCode":         url.StatusCode,
				})
			}

		case err := <-errChan:
			// Отправляем ошибку по SSE; 
			// если хотим прервать процесс целиком — можно сделать return
			sendSSEEvent(c, "error", gin.H{"error": err.Error()})

		case progress := <-progressChan:
			sendSSEProgress(c, progress, false)

			if progress == "100" {
				// Запускаем активное сканирование, если необходимо
				_ = aScan.StartActiveScan()

				// Закрываем горутину пассивного сканирования
				close(doneChan)
			}

		case <-doneChan:
			return

		case <-ticker.C:
			// Раз в 250мс «пульсируем»: в данном случае ничего не делаем,
			// но можем использовать для «keep-alive» (например, отправить ping).
			// sendSSEEvent(c, "ping", "keep-alive")
		}
	}
}

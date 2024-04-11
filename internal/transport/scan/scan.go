package scan

import (
	"github.com/YoungGoofy/gozap/pkg/gozap"
	"github.com/gin-gonic/gin"
	"net/http"
	"sync"
)

type Scanner struct {
	MainScanner gozap.MainScan
}

func NewScanner(apiKey string) *Scanner {
	newScan := gozap.NewMainScan("", apiKey)
	//newSpider := gozap.NewSpider(*newScan)
	return &Scanner{*newScan}
}

func (s *Scanner) StartScan(c *gin.Context) {
	newUrl := struct {
		Url string `form:"url"`
	}{}
	var wg sync.WaitGroup
	if err := c.ShouldBind(&newUrl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.MainScanner.AddUrl(newUrl.Url)
	spider := gozap.NewSpider(s.MainScanner)
	if err := spider.GetSessionId(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// Отправляем начальное состояние сканирования
	ssEventStatus(c, "0", false)

	checkStatus(c, spider, &wg)

	ssEventStatus(c, "100", true)
}

func ssEventStatus(c *gin.Context, progressPercentage string, completed bool) {
	c.SSEvent("progress", map[string]interface{}{
		"progressPercentage": progressPercentage,
		"completed":          completed,
	})
	c.Writer.Flush()
}

func checkStatus(c *gin.Context, spider *gozap.Spider, wg *sync.WaitGroup) {
	dataCh := make(chan gozap.UrlsInScope)
	errCh := make(chan error)
	statusCh := make(chan string)
	done := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		spider.AsyncGetResult(dataCh, errCh, statusCh, done)
	}()
	for {
		select {
		case urls := <-dataCh:
			for _, url := range urls {
				c.SSEvent("results", map[string]string{
					"processed":          url.Processed,
					"statusReason":       url.StatusReason,
					"method":             url.Method,
					"reasonNotProcessed": url.ReasonNotProcessed,
					"messageId":          url.MessageID,
					"url":                url.URL,
					"statusCode":         url.StatusCode,
				})
				c.Writer.Flush()
			}
		case err := <-errCh:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
		case status := <-statusCh:
			ssEventStatus(c, status, false)
			if status == "100" {
				close(done)
			}
		case <-done:
			wg.Wait()
			close(dataCh)
			close(errCh)
			close(statusCh)
			return
		}
	}
}

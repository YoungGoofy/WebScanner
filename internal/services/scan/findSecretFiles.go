package scan

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

type Config struct {
    BaseURL          string   `yaml:"baseURL"`
    Paths            []string `yaml:"paths"`
    TimeoutSeconds   int      `yaml:"timeoutSeconds"`
    ValidStatusCodes []int    `yaml:"validStatusCodes"`
    Verbose          bool     `yaml:"verbose"`
}

func checkFile(cfg Config, baseUrl string, path string) (bool, error) {
    baseURL := strings.TrimRight(baseUrl, "/")
    targetURL := baseURL + "/" + path

    // Создаём HTTP-клиент с таймаутом из конфига
    client := &http.Client{
        Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
    }

    resp, err := client.Get(targetURL)
    if err != nil {
        return false, err
    }
    defer resp.Body.Close()

    // Проверяем, входит ли статус-код в список cfg.ValidStatusCodes
    for _, code := range cfg.ValidStatusCodes {
        if resp.StatusCode == code {
            return true, nil
        }
    }

    return false, nil
}

func scanForForgottenFiles(cfg Config, baseUrl string) []string {
    var discovered []string

    for _, p := range cfg.Paths {
        found, err := checkFile(cfg, baseUrl, p)
        if err != nil {
            // Логируем ошибку, если verbose = true
            if cfg.Verbose {
                log.Printf("Ошибка при проверке %s: %v\n", p, err)
            }
            continue
        }

        if found {
            discovered = append(discovered, p)
        } else if cfg.Verbose {
            log.Printf("[VERBOSE] Файл '%s' недоступен или отсутствует.\n", p)
        }
    }

    return discovered
}

func scanSecretFiles(c *gin.Context, baseUrl string) {
	// 1. Читаем конфигурационный файл
    configData, err := ioutil.ReadFile("internal/configs/secretFiles.yaml")
    if err != nil {
        log.Fatalf("Не удалось прочитать файл config.yaml: %v", err)
    }

    // 2. Парсим YAML в структуру Config
    var cfg Config
    if err := yaml.Unmarshal(configData, &cfg); err != nil {
        log.Fatalf("Не удалось распарсить config.yaml: %v", err)
    }

    // 3. Если в конфиге не задан timeoutSeconds, ставим значение по умолчанию
    if cfg.TimeoutSeconds <= 0 {
        cfg.TimeoutSeconds = 5 // 5 секунд по умолчанию
    }

    // 4. Если список ValidStatusCodes пуст, подставим стандартные (200, 302)
    if len(cfg.ValidStatusCodes) == 0 {
        cfg.ValidStatusCodes = []int{200, 302}
    }

    // 5. Запускаем сканирование
    discovered := scanForForgottenFiles(cfg, baseUrl)

	id := 0
    // 6. Выводим результат
    if len(discovered) == 0 {
        sendSSEEvent(c, "secretFiles", map[string]any{
			"id": id,
			"file": "Ничего не найдено.",
		})
    } else {
        fmt.Println("Найдены следующие потенциально «забытые» файлы:")
        for _, file := range discovered {
            sendSSEEvent(c, "secretFiles", map[string]any{
				"id": id,
				"file": file,
			})
			id++
        }
    }
}
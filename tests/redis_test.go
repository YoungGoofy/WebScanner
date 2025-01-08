package tests__test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/YoungGoofy/WebScanner/internal/db"
	"github.com/YoungGoofy/WebScanner/internal/db/crud"
	"github.com/YoungGoofy/WebScanner/internal/db/models"
	"github.com/go-redis/redis/v8"
)

var ctxTest = context.Background()

func TestNewRedisClient(t *testing.T) {
    // Допустим, у нас есть локальный Redis на порту 6379, без пароля, DB=0
    cfg := &db.RedisConfig{
        Host:     "localhost",
        Port:     6379,
        Password: "",
        DB:       0,
    }

    // Пытаемся создать и инициализировать клиента
    client, err := db.NewRedisClient(cfg)
    if err != nil {
        t.Fatalf("Ошибка при подключении к Redis: %v", err)
    }

    // Дополнительная проверка, посылаем PING
    if pingErr := client.Ping(ctxTest).Err(); pingErr != nil {
        t.Fatalf("Ошибка при отправке PING: %v", pingErr)
    }

    // Проверяем, что клиент действительно работает: запишем и прочитаем ключ
    testKey := "test_key"
    testValue := "test_value"

    if err := client.Set(ctxTest, testKey, testValue, 0).Err(); err != nil {
        t.Fatalf("Не удалось записать ключ в Redis: %v", err)
    }

    val, err := client.Get(ctxTest, testKey).Result()
    if err != nil {
        t.Fatalf("Не удалось прочитать ключ из Redis: %v", err)
    }
    if val != testValue {
        t.Errorf("Значение не совпадает. Ожидалось %q, а получили %q", testValue, val)
    }

    // Убираем за собой
    if err := client.Del(ctxTest, testKey).Err(); err != nil {
        t.Logf("Не удалось удалить тестовый ключ: %v", err)
    }

    // Закрываем соединение
    _ = client.Close()
}

func newTestRedisClient() *redis.Client {
    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
        DB:   0,
        // Password, если нужно
    })
    return rdb
}

func TestInsertActiveScan(t *testing.T) {
    rdb := newTestRedisClient()
    defer rdb.Close()

    // Для чистоты теста можно очистить DB
    if err := rdb.FlushDB(ctxTest).Err(); err != nil {
        t.Fatalf("Не удалось очистить Redis перед тестом: %v", err)
    }

    // Подготавливаем тестовые данные
    input := models.ActiveScan{
        ID:          "test123",
        Name:        "Test Scan",
        CweID:       "CWE-79",
        Risk:        "High",
        Method:      "GET",
        Link:        "http://example.com/vuln",
        Description: "XSS vulnerability",
        Solution:    "Escape user input",
    }

    // Вызываем функцию InsertActiveScan
    if err := crud.InsertActiveScan(rdb, input); err != nil {
        t.Fatalf("Ошибка InsertActiveScan: %v", err)
    }

    // Считываем сырые данные из Redis по сформированному ключу
    key := fmt.Sprintf("active_scan:%s", input.ID)
    raw, err := rdb.Get(ctxTest, key).Result()
    if err != nil {
        t.Fatalf("Ошибка чтения ключа %q из Redis: %v", key, err)
    }

    // Десериализуем обратно в структуру
    var got models.ActiveScan
    if err := json.Unmarshal([]byte(raw), &got); err != nil {
        t.Fatalf("Ошибка при десериализации JSON: %v", err)
    }

    // Сравниваем поля
    if !reflect.DeepEqual(input, got) {
        t.Errorf("Сохранённая структура ActiveScan не совпадает.\nОжидалось: %+v\nПолучено: %+v", input, got)
    }
}

// -----------------------------------------------------------------------------
// Тест для InsertStatusWork

func TestInsertStatusWork(t *testing.T) {
    rdb := newTestRedisClient()
    defer rdb.Close()

    // Очистим БД
    if err := rdb.FlushDB(ctxTest).Err(); err != nil {
        t.Fatalf("Не удалось очистить Redis перед тестом: %v", err)
    }

    // Подготавливаем тестовые данные
    input := models.StatusWork{
        ActiveScan:  "running",
        PassiveScan: "idle",
        FinalResult: "not_ready",
    }

    // Вызываем функцию InsertStatusWork
    if err := crud.InsertStatusWork(rdb, input); err != nil {
        t.Fatalf("Ошибка InsertStatusWork: %v", err)
    }

    // Ключ фиксированный: "status_work"
    raw, err := rdb.Get(ctxTest, "status_work").Result()
    if err != nil {
        t.Fatalf("Ошибка чтения ключа %q из Redis: %v", "status_work", err)
    }

    // Десериализуем
    var got models.StatusWork
    if err := json.Unmarshal([]byte(raw), &got); err != nil {
        t.Fatalf("Ошибка при десериализации JSON: %v", err)
    }

    // Сравниваем
    if !reflect.DeepEqual(input, got) {
        t.Errorf("Сохранённая структура StatusWork не совпадает.\nОжидалось: %+v\nПолучено: %+v", input, got)
    }
}

// -----------------------------------------------------------------------------
// Тест для InsertPassiveScan

func TestInsertPassiveScan(t *testing.T) {
    rdb := newTestRedisClient()
    defer rdb.Close()

    // Очистим БД
    if err := rdb.FlushDB(ctxTest).Err(); err != nil {
        t.Fatalf("Не удалось очистить Redis перед тестом: %v", err)
    }

    // Подготавливаем тестовые данные
    input := models.PassiveScan{
        ID:               42,  // предположим, что поле ID есть в структуре, раз key = passive_scan:%d
        Processed:        "yes",
        StatusReason:     "OK",
        Method:           "GET",
        ReasonNotProcessed: "",
        MessageId:        "msg-123",
        Link:             "http://example.com/passive",
        StatusCode:       "200",
    }

    // Вызываем функцию InsertPassiveScan
    if err := crud.InsertPassiveScan(rdb, input); err != nil {
        t.Fatalf("Ошибка InsertPassiveScan: %v", err)
    }

    // Формируем ключ "passive_scan:<ID>"
    key := fmt.Sprintf("passive_scan:%d", input.ID)
    raw, err := rdb.Get(ctxTest, key).Result()
    if err != nil {
        t.Fatalf("Ошибка чтения ключа %q из Redis: %v", key, err)
    }

    // Десериализуем
    var got models.PassiveScan
    if err := json.Unmarshal([]byte(raw), &got); err != nil {
        t.Fatalf("Ошибка при десериализации JSON: %v", err)
    }

    // Сравниваем
    if !reflect.DeepEqual(input, got) {
        t.Errorf("Сохранённая структура PassiveScan не совпадает.\nОжидалось: %+v\nПолучено: %+v", input, got)
    }
}
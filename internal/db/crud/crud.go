package crud

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/YoungGoofy/WebScanner/internal/db/models"
	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

// CreateDB очищает выбранную базу
func CreateDB(rdb *redis.Client) error {
    err := rdb.FlushDB(ctx).Err()
    if err != nil {
        return fmt.Errorf("ошибка при очистке Redis DB: %w", err)
    }
    return nil
}

// ClearDB — отдельная функция, если будет нужно вызывать очистку в другом месте
func ClearDB(rdb *redis.Client) error {
    return CreateDB(rdb)
}

// InsertActiveScan записывает структуру ActiveScan
func InsertActiveScan(rdb *redis.Client, scan models.ActiveScan) error {
    data, err := json.Marshal(scan)
    if err != nil {
        return fmt.Errorf("ошибка при сериализации ActiveScan: %w", err)
    }

    key := fmt.Sprintf("active_scan:%s", scan.ID)

    err = rdb.Set(ctx, key, data, 0).Err()
    if err != nil {
        return fmt.Errorf("ошибка при записи ActiveScan в Redis: %w", err)
    }

    return nil
}

func InsertStatusWork(rdb *redis.Client, status models.StatusWork) error {
    data, err := json.Marshal(status)
    if err != nil {
        return fmt.Errorf("ошибка при сериализации StatusWork: %w", err)
    }

    const key = "status_work" // Фиксированный ключ для единственного статуса
    if err := rdb.Set(ctx, key, data, 0).Err(); err != nil {
        return fmt.Errorf("ошибка при записи StatusWork в Redis: %w", err)
    }

    return nil
}

func InsertPassiveScan(rdb *redis.Client, scan models.PassiveScan) error {
    data, err := json.Marshal(scan)
    if err != nil {
        return fmt.Errorf("ошибка при сериализации PassiveScan: %w", err)
    }

    key := fmt.Sprintf("passive_scan:%d", scan.ID)
    if err := rdb.Set(ctx, key, data, 0).Err(); err != nil {
        return fmt.Errorf("ошибка при записи PassiveScan в Redis: %w", err)
    }

    return nil
}
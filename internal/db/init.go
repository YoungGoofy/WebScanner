package db

import (
    "context"
    "fmt"

    "github.com/go-redis/redis/v8"
    "github.com/pelletier/go-toml"
)

// RedisConfig - структура для хранения параметров Redis
type RedisConfig struct {
    Host     string `toml:"host"`
    Port     int    `toml:"port"`
    Password string `toml:"password"`
    DB       int    `toml:"db"`
}

func LoadConfigToml(path string) (*RedisConfig, error) {
    // Загружаем томл-файл
    configTree, err := toml.LoadFile(path)
    if err != nil {
        return nil, fmt.Errorf("не удалось прочитать файл конфигурации: %w", err)
    }

    // Считываем секцию "database" и маппим в RedisConfig
    databaseTree := configTree.Get("database")
    if databaseTree == nil {
        return nil, fmt.Errorf("не найдена секция [database] в конфигурационном файле")
    }

    // Преобразуем в структуру RedisConfig
    var redisCfg RedisConfig
    err = configTree.Get("database").(*toml.Tree).Unmarshal(&redisCfg)
    if err != nil {
        return nil, fmt.Errorf("ошибка при десериализации настроек Redis: %w", err)
    }

    return &redisCfg, nil
}

var ctx = context.Background()

func NewRedisClient(cfg *RedisConfig) (*redis.Client, error) {
    rdb := redis.NewClient(&redis.Options{
        Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
        Password: cfg.Password, // можно оставить пустым, если пароль не установлен
        DB:       cfg.DB,       // номер базы (по умолчанию 0)
    })

    // Проверяем соединение, отправив команду PING
    if err := rdb.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("ошибка при подключении к Redis: %w", err)
    }

    return rdb, nil
}
package main

import (
    "fmt"
    "net/http"
)

// testFiles – «забытые» файлы, которые сервер будет считать существующими.
// Ключ – это путь (например, "/.env"), значение – это контент, который вернётся при запросе.
var testFiles = map[string]string{
    "/.env":             "DB_USER=root\nDB_PASSWORD=supersecret",
    "/backup.sql":       "-- SQL dump content",
    "/config.bak":       "some config data here",
    "/wp-config.php.bak": "<?php echo 'wp config backup'; ?>",
}

// Если вы хотите протестировать редиректы (например, код 302), 
// можно добавить ещё один специальный путь:
var redirectPaths = map[string]string{
    "/old-backup.zip": "/new-backup.zip", // При запросе /old-backup.zip сервер сделает 302 на /new-backup.zip
}

// handleRequest – единая функция-обработчик всех входящих запросов.
func handleRequest(w http.ResponseWriter, r *http.Request) {
    path := r.URL.Path

    // 1. Если путь встречается среди "редиректов" – шлём 302 Found
    if newLocation, ok := redirectPaths[path]; ok {
        http.Redirect(w, r, newLocation, http.StatusFound)
        return
    }

    // 2. Если путь встречается среди «забытых» файлов – шлём 200 OK + содержимое
    if content, ok := testFiles[path]; ok {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte(content))
        return
    }

    // 3. Для всего остального возвращаем 404 Not Found
    http.NotFound(w, r)
}

func main() {
    // Регистрируем обработчик на все пути
    http.HandleFunc("/", handleRequest)

    // Запускаем сервер на порту 8080
    fmt.Println("Запуск тестового сервера на :3001 ...")
    err := http.ListenAndServe(":3001", nil)
    if err != nil {
        fmt.Printf("Ошибка запуска сервера: %v\n", err)
    }
}

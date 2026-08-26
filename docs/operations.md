# Operations — Mailbridge

## Сборка

```bash
make build
```

Собирает фронтенд (Vite), копирует статику в `cmd/mailbridge/static/`, компилирует Go-бинарник в `build/mailbridge`.

## Запуск

### Production

```bash
cp configs/config.example.env configs/config.env
# отредактировать
make run
```

### Dev

```bash
# Бекенд на 8081
make run-dev

# Фронтенд на 5173 (прокси на 8081)
cd frontend && npm run dev
```

## Обязательные переменные

| Переменная | Описание | Пример |
|-----------|----------|--------|
| MAILBRIDGE_IMAP_SERVER | IMAP-сервер | mail.example.com |
| MAILBRIDGE_IMAP_USER | Логин | support@example.com |
| MAILBRIDGE_IMAP_PASS | Пароль | |
| MAILBRIDGE_SMTP_SERVER | SMTP-сервер | smtp.example.com |
| MAILBRIDGE_SMTP_FROM | Отправитель | support@example.com |
| MAILBRIDGE_AUTH_USER | Логин веб-интерфейса | admin |
| MAILBRIDGE_AUTH_PASS | Пароль веб-интерфейса | |

## Health-check

```bash
curl http://localhost:8080/health
# → {"status":"ok"}

curl http://localhost:8080/ready
# → {"checks":{"database":"ok","imap":"ok"},"status":true}
```

## Логи

JSON-формат при `MAILBRIDGE_LOG_FORMAT=json`. Ключевые сообщения:

- `inbox item created` — новое входящее
- `inbox item queued for AI processing` — в очереди
- `[AI] Ответ LLM` — ответ модели
- `[AIWorker] inbox item processed successfully` — обработано

## Метрики

`GET /metrics` — Prometheus-формат:
- `mailbridge_emails_processed_total`
- `mailbridge_issues_created_total`
- `mailbridge_imap_connected`
- `mailbridge_plane_available`

## Частые проблемы

### IMAP переподключение

При обрыве — автоматический reconnect с backoff. Лог: `attempting IMAP reconnect`.

### Дедупликация вложений

Файлы хранятся по SHA-256 в `data/attachments/{hash[0:2]}/{hash[2:4]}/{hash}`. Одинаковые файлы не дублируются.

### AI-очередь

При перезапуске необработанные (`ai_processed = 0`) автоматически загружаются в очередь.

### SQLite WAL

Файлы `-shm` и `-wal` в `data/` — нормальное состояние WAL. Не удалять при работающем сервисе.

## Бэкап

```bash
# Остановить сервис
systemctl --user stop mailbridge

# Скопировать
tar czf backup-$(date +%Y%m%d).tar.gz data/

# Запустить
systemctl --user start mailbridge
```

## Деплой (systemd user)

```ini
# ~/.config/systemd/user/mailbridge.service
[Unit]
Description=Mailbridge

[Service]
Type=simple
WorkingDirectory=%h/apps/mailbridge
ExecStart=%h/apps/mailbridge/bin/mailbridge
EnvironmentFile=%h/apps/mailbridge/configs/config.env
Restart=always

[Install]
WantedBy=default.target
```

```bash
systemctl --user daemon-reload
systemctl --user enable mailbridge
systemctl --user start mailbridge
loginctl enable-linger $USER
```
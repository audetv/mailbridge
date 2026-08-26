# Security

## Секреты

- `configs/config.env` — НЕ коммитить (в .gitignore)
- Образец: `configs/config.example.env`
- Секреты только через env-переменные

## Аутентификация

- JWT-подобный токен: `token-USERNAME-YYYYMMDD`
- Учётные данные: `MAILBRIDGE_AUTH_USER` / `MAILBRIDGE_AUTH_PASS`
- Секрет для подписи: `MAILBRIDGE_AUTH_SECRET`

## Вложения

- Файлы хранятся по SHA-256, не по имени
- Path traversal защищён в `GetAttachment`
- Дедупликация — одинаковые файлы хранятся один раз

## Что не коммитить

- `data/` — БД, вложения, отладка
- `configs/config.env` — секреты
- `cmd/mailbridge/static/` — артефакты сборки

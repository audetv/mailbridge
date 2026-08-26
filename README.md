# Mailbridge

Сервис для управления входящими обращениями из электронной почты. Превращает письма в задачи с помощью AI-анализа.

## Возможности

- **Лента входящих** — все письма в едином потоке с фильтрами и статусами
- **AI-анализ** — автоматическое распознавание задач, обновлений, выполненной работы
- **Задачи** — создаются из писем, с приоритетами, проектами, исполнителями
- **Вложения** — все типы файлов (DOCX, XLSX, PDF, RTF, ICS, PPTX, изображения)
- **Дедупликация файлов** — по SHA-256, файл хранится один раз
- **Веб-интерфейс** — Vue 3 + PrimeVue, тёмная/светлая тема
- **Живое обновление** — WebSocket, счётчики, toast-уведомления
- **Очередь AI** — асинхронная обработка с retry и backoff

## Быстрый старт

### Требования

- Go 1.26+
- Node.js 20+ (для сборки фронтенда)
- Ollama (опционально, для AI-анализа)

### Сборка

```bash
make build
```

### Настройка

```bash
cp configs/config.example.env configs/config.env
nano configs/config.env
```

Обязательные параметры:
- `MAILBRIDGE_IMAP_SERVER`, `MAILBRIDGE_IMAP_USER`, `MAILBRIDGE_IMAP_PASS`
- `MAILBRIDGE_SMTP_SERVER`, `MAILBRIDGE_SMTP_FROM`
- `MAILBRIDGE_AUTH_USER`, `MAILBRIDGE_AUTH_PASS`

Для AI:
- `MAILBRIDGE_AI_ENABLED=true`
- `MAILBRIDGE_AI_PROVIDER=ollama` или `openai`
- `MAILBRIDGE_AI_BASE_URL=http://localhost:11434`
- `MAILBRIDGE_AI_MODEL=email-assistant-v2`

### Запуск

```bash
make run
```

Откройте http://localhost:8080

## Разработка

### Dev-режим

Бекенд:
```bash
make run-dev  # порт 8081
```

Фронтенд:
```bash
cd frontend
npm install
npm run dev  # порт 5173, прокси на 8081
```

### Тесты

```bash
make test
make lint
```

## Документация

- [Архитектура](docs/ARCHITECTURE.md)
- [Модель данных](docs/data-model.md)
- [REST API](docs/api.md)
- [AI-конвейер](docs/ai-pipeline.md)
- [Эксплуатация](docs/operations.md)
- [Архитектурные решения (ADR)](docs/adr/)

## Агентный контекст

Для AI-агентов: [AGENTS.md](AGENTS.md) — команды, карта репозитория, правила и инварианты.

## Лицензия

MIT
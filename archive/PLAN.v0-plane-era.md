# Проект: Mailbridge

**Описание:** Интерактивный почтовый шлюз между email и Plane, с NLP-классификацией, обработкой вложений, двухсторонней синхронизацией и гарантированной доставкой.

---

## План разработки (12 этапов)

```
Этап 0: Инициализация проекта и CI
Этап 1: Конфигурация и Graceful Shutdown
Этап 2: Логирование и Health Checks
Этап 3: Store — интерфейс и SQLite-реализация
Этап 4: NLP — токенизатор, стеммер, n-граммы
Этап 5: Classifier — Rule-Based с NLP и urgency booster
Этап 6: Extractor — MIME-парсинг, очистка тела, вложения
Этап 7: Field Parser — извлечение ключей из тела
Этап 8: Plane Client — REST API клиент
Этап 9: Message Processor — оркестрация обработки письма
Этап 10: Mail Reader — IMAP-клиент и Inbound Worker
Этап 11: Email Sender, Outbox, Outbound Worker
Этап 12: Webhook Handler — приём событий из Plane
```

---

Сейчас создам файл `PLAN.md` с детальным описанием каждого этапа: конкретные файлы, интерфейсы, структуры, тесты, критерии приёмки.

```markdown
# Mailbridge — План разработки

## Этап 0: Инициализация проекта и CI

### Цель
Создать структуру проекта, модуль, зависимости, базовый CI.

### Действия
- `go mod init github.com/your-org/mailbridge`
- Создать структуру директорий
- Добавить зависимости: spdlog, sqlite3, vmime/go-imap, gomail, snowball
- Настроить golangci-lint
- Настроить GitHub Actions: lint, test
- Создать Makefile: build, test, lint, run
- Написать тест-заглушку для проверки CI

### Критерии приёмки
- `make lint` проходит без ошибок
- `make test` проходит (1 заглушка)
- CI зелёный в репозитории

---

## Этап 1: Конфигурация и Graceful Shutdown

### Цель
Загрузка конфигурации из .env и аргументов командной строки. Корректное завершение по SIGINT/SIGTERM.

### Файлы
- `internal/config/config.go` — структура Config, метод Load()
- `internal/config/config_test.go` — тесты загрузки
- `internal/app/app.go` — структура App, методы Run(), Shutdown()
- `internal/app/signals.go` — обработка сигналов ОС
- `cmd/mailbridge/main.go` — точка входа

### Интерфейсы
```go
type Config struct {
    IMAP     IMAPConfig
    SMTP     SMTPConfig
    Plane    PlaneConfig
    Webhook  WebhookConfig
    Storage  StorageConfig
    NLP      NLPConfig
    Logging  LoggingConfig
}

type App struct {
    Config    *Config
    Components []Component  // интерфейс: Start(ctx) error; Stop(ctx) error
}
```

### Критерии приёмки
- Приложение запускается и ждёт SIGINT
- При SIGINT вызывает Stop() у всех компонентов
- Конфигурация валидируется (обязательные поля)
- Тесты: загрузка из .env, валидация, значения по умолчанию

---

## Этап 2: Логирование и Health Checks

### Цель
Структурированное логирование с уровнями. HTTP-сервер с /health и /ready.

### Файлы
- `internal/logging/logger.go` — обёртка над slog
- `internal/logging/logger_test.go`
- `internal/health/server.go` — HTTP-сервер health check
- `internal/health/checks.go` — проверки компонентов
- `internal/health/server_test.go`

### Интерфейсы
```go
type HealthChecker interface {
    Name() string
    Check(ctx context.Context) error
}

type HealthServer struct {
    server   *http.Server
    checks   []HealthChecker
}
```

### Endpoints
- `GET /health` — 200 OK всегда (liveness)
- `GET /ready` — 200 если все Check() прошли (readiness)
- `GET /metrics` — Prometheus метрики (заглушка)

### Критерии приёмки
- Логи пишутся в JSON при LOG_FORMAT=json
- Уровни: debug, info, warn, error
- /health отвечает 200
- /ready отвечает 503 если БД недоступна
- Тесты: проверка форматов, уровней

---

## Этап 3: Store — интерфейс и SQLite-реализация

### Цель
Абстрактный интерфейс хранилища. SQLite-реализация с миграциями. Готовность к Postgres в будущем.

### Файлы
- `internal/store/store.go` — интерфейс Store
- `internal/store/models.go` — структуры EmailMapping, ReplyLog, OutboxItem
- `internal/store/sqlite/sqlite.go` — SQLite-реализация
- `internal/store/sqlite/migrations.go` — SQL-миграции
- `internal/store/sqlite/sqlite_test.go` — интеграционные тесты

### Интерфейс
```go
type Store interface {
    Migrate(ctx context.Context) error
    SaveMapping(ctx context.Context, m *EmailMapping) error
    GetMappingByMessageID(ctx context.Context, msgID string) (*EmailMapping, error)
    GetLatestMappingByIssueID(ctx context.Context, issueID string) (*EmailMapping, error)
    MessageExists(ctx context.Context, msgID string) (bool, error)
    FindMappingByReferences(ctx context.Context, refs []string) (*EmailMapping, error)
    SaveReplyLog(ctx context.Context, log *ReplyLog) error
    ReplyExists(ctx context.Context, msgID string) (bool, error)
    EnqueueOutbox(ctx context.Context, payload string) error
    GetPendingOutbox(ctx context.Context, limit int) ([]*OutboxItem, error)
    MarkOutboxSent(ctx context.Context, id int64) error
    MarkOutboxFailed(ctx context.Context, id int64, errMsg string) error
    Close() error
}
```

### Таблицы
- `email_mapping` — связь Message-ID ↔ Plane Issue ID
- `reply_log` — история отправленных ответов
- `outbox` — очередь исходящих писем

### Критерии приёмки
- Миграции создают таблицы в новой БД
- SaveMapping + GetMappingByMessageID работают
- MessageExists возвращает true для сохранённого
- FindMappingByReferences находит по массиву refs
- Outbox: Enqueue → GetPending → MarkSent
- Все методы покрыты тестами с реальной SQLite в памяти

---

## Этап 4: NLP — токенизатор, стеммер, n-граммы

### Цель
Пакет нормализации текста для русского и английского языков.

### Файлы
- `internal/classifier/nlp/tokenizer.go`
- `internal/classifier/nlp/stemmer.go`
- `internal/classifier/nlp/ngram.go`
- `internal/classifier/nlp/nlp_test.go`

### Интерфейсы
```go
type Tokenizer struct {
    stopWords map[string]bool
}
func (t *Tokenizer) Tokenize(text string) []string

type Stemmer interface {
    Stem(word string) string
}
type RussianStemmer struct{}
type EnglishStemmer struct{}

type NGramGenerator struct{}
func (g *NGramGenerator) Generate(tokens []string) []string // уни-, би-, триграммы
```

### Алгоритм стеммера
- RussianStemmer: отсечение окончаний и суффиксов по правилам
- EnglishStemmer: обёртка над porterstemmer

### Критерии приёмки
- Токенизатор удаляет стоп-слова и знаки препинания
- "не работает сайт" → токены ["работает", "сайт"] (без "не" — оставить значимое)
- Стеммер: "работает" → "работа", "ошибки" → "ошибк"
- N-граммы: ["ошибк", "сервер"] → ["ошибк", "сервер", "ошибк_сервер"]
- Тесты: 15+ кейсов для каждого компонента

---

## Этап 5: Classifier — Rule-Based с NLP и Urgency Booster

### Цель
Классификатор, принимающий текст и возвращающий проект, тип, приоритет с confidence.

### Файлы
- `internal/classifier/classifier.go` — интерфейс
- `internal/classifier/rule_based.go` — RuleBasedClassifier
- `internal/classifier/matcher.go` — Matcher с весами
- `internal/classifier/urgency.go` — UrgencyBooster
- `internal/classifier/rules.go` — дефолтные правила
- `internal/classifier/classifier_test.go`

### Интерфейс
```go
type Classifier interface {
    Classify(ctx context.Context, text string, projects, types []string) (*Classification, error)
}

type Classification struct {
    Project     string
    Type        string
    Priority    string
    Confidence  float64
    NeedsTriage bool
}
```

### Urgency Booster
```go
type UrgencyBooster struct {
    patterns []string
}
func (b *UrgencyBooster) Boost(text string, currentPriority string) string
// Если найдены срочные слова → priority = "urgent"
```

### Логика Matcher
- Токенизация текста → стемминг → n-граммы
- Сравнение с нормализованными правилами
- Подсчёт весов: униграмма=1, биграмма=3, триграмма=5
- Умножение на вес правила
- Выбор лучшего совпадения по каждому полю

### Критерии приёмки
- "Не открывается кабинет арендатора, ошибка 500" → project=ТРК, type=bug, priority=high
- "Срочно! Сайт отеля упал!" → priority=urgent (booster)
- "Добавьте баннер на сайт трк" → type=content
- "Нужен доступ к админке театра" → type=access, project=Театр
- Пустой текст → NeedsTriage=true
- Тесты: 20+ классификационных кейсов

---

## Этап 6: Extractor — MIME-парсинг, очистка тела, вложения

### Цель
Извлечение текста, вложений из сырого email. Очистка от истории переписки и подписей.

### Файлы
- `internal/extractor/extractor.go`
- `internal/extractor/cleaner.go`
- `internal/extractor/attachments.go`
- `internal/extractor/extractor_test.go`

### Интерфейсы
```go
type ExtractedEmail struct {
    MessageID   string
    From        string
    To          string
    Subject     string
    BodyText    string
    BodyHTML    string
    References  []string
    InReplyTo   string
    Attachments []Attachment
    ReceivedAt  time.Time
}

type Attachment struct {
    Filename    string
    ContentType string
    Size        int64
    Data        []byte
    StoragePath string  // путь после сохранения
}

type AttachmentStore interface {
    Save(ctx context.Context, attachment *Attachment) error
    Get(ctx context.Context, path string) (*Attachment, error)
}
```

### Компоненты
- `Extractor.Extract(raw []byte)` → ExtractedEmail
- `Cleaner.CleanBody(text string)` → очищенный текст
- `AttachmentStore.Save()` → сохраняет в filesystem/S3/MinIO

### Алгоритм очистки
1. Удаление цитируемого ответа (маркеры: "-----Original Message-----", "писал(а):")
2. Удаление подписей (разделитель "-- ")
3. Удаление пустых строк в конце

### Критерии приёмки
- Извлекает текст из multipart/alternative
- Извлекает HTML и конвертирует в текст
- Вложения сохраняются в указанную директорию
- Имена файлов санитизируются (удаление спецсимволов)
- Тест с реальным .eml файлом

---

## Этап 7: Field Parser — извлечение ключей из тела

### Цель
Парсинг тела письма на наличие структурированных полей.

### Файлы
- `internal/parser/parser.go`
- `internal/parser/parser_test.go`

### Интерфейс
```go
type ParsedFields struct {
    Project     string
    Type        string
    Priority    string
    Deadline    string
    Assignee    string
    Body        string  // текст без полей
    HasFields   bool
}

type FieldParser struct{}
func (p *FieldParser) Parse(body string) *ParsedFields
```

### Поддерживаемые ключи
- Проект:, Project:
- Тип:, Type:
- Приоритет:, Priority:
- Дедлайн:, Deadline:
- Исполнитель:, Assignee:

### Критерии приёмки
- Парсит "Проект: ТРК" → Project="ТРК"
- Мультиязычность: "Project: TRK" → Project="TRK"
- Не ломается на отсутствии полей
- Body содержит только текст после полей
- Тесты: 10+ кейсов

---

## Этап 8: Plane Client — REST API клиент

### Цель
Клиент для взаимодействия с Plane API.

### Файлы
- `internal/plane/client.go`
- `internal/plane/types.go`
- `internal/plane/client_test.go`

### Методы
```go
type PlaneClient struct {
    baseURL string
    apiKey  string
    http    *http.Client
}

func (c *PlaneClient) CreateIssue(ctx context.Context, req *CreateIssueRequest) (*Issue, error)
func (c *PlaneClient) GetIssue(ctx context.Context, id string) (*Issue, error)
func (c *PlaneClient) AddComment(ctx context.Context, issueID, body string) (*Comment, error)
func (c *PlaneClient) GetProjects(ctx context.Context) ([]Project, error)
func (c *PlaneClient) GetLabels(ctx context.Context, projectID string) ([]Label, error)
```

### Особенности
- Retry с exponential backoff (3 попытки)
- Таймаут запроса: 10 секунд
- Обработка 429 (rate limit)

### Критерии приёмки
- CreateIssue создаёт задачу в тестовом Plane
- AddComment добавляет комментарий
- GetProjects возвращает список проектов
- Retry работает при ошибках сети
- Тесты с httptest.NewServer (mock Plane API)

---

## Этап 9: Message Processor — оркестрация обработки письма

### Цель
Центральный оркестратор, определяющий действие для входящего письма.

### Файлы
- `internal/processor/processor.go`
- `internal/processor/processor_test.go`

### Логика
```go
type MessageProcessor struct {
    store      Store
    classifier Classifier
    extractor  *Extractor
    parser     *FieldParser
    planeClient *PlaneClient
}

func (p *MessageProcessor) Process(ctx context.Context, rawEmail []byte) (*ProcessResult, error)
```

### Сценарии
1. **Новое письмо с ID задачи в теме** (`[WEB-123]`) → добавить комментарий
2. **Новое письмо, References указывают на известную задачу** → добавить комментарий
3. **Новое письмо, без связи с задачей** → создать задачу
4. **Дубликат Message-ID** → игнорировать

### ProcessResult
```go
type ProcessResult struct {
    Action        ActionType // CREATE_ISSUE, ADD_COMMENT, IGNORE
    IssueID       string
    IssueSequence string
    Mapping       *EmailMapping
    Error         error
}
```

### Критерии приёмки
- Все 4 сценария покрыты тестами
- Создание задачи: вызывается Classifier + PlaneClient.CreateIssue
- Комментарий: вызывается PlaneClient.AddComment
- Дубликат: возвращается IGNORE
- Мок хранилища, классификатора, API

---

## Этап 10: Mail Reader — IMAP-клиент и Inbound Worker

### Цель
Подключение к IMAP, сканирование новых писем, передача в Processor.

### Файлы
- `internal/mailbox/reader.go`
- `internal/mailbox/reader_test.go`
- `internal/worker/inbound.go`

### Интерфейс
```go
type MailReader struct {
    imapClient *imapclient.Client
    config     IMAPConfig
}

func (r *MailReader) Connect(ctx context.Context) error
func (r *MailReader) FetchUnseen(ctx context.Context) ([]*RawEmail, error)
func (r *MailReader) MarkProcessed(ctx context.Context, uid uint32) error
func (r *MailReader) MarkErrored(ctx context.Context, uid uint32) error
```

### Inbound Worker
```go
type InboundWorker struct {
    reader    *MailReader
    processor *MessageProcessor
    interval  time.Duration
}
// Бесконечный цикл: FetchUnseen → Process → MarkProcessed
```

### Критерии приёмки
- Подключается к тестовому IMAP (можно greenmail/mailhog)
- Получает непрочитанные письма
- Помечает прочитанными после обработки
- Перемещает в Archive при успехе, в Errors при ошибке
- Inbound Worker работает по тикеру
- Graceful shutdown: завершает текущую итерацию

---

## Этап 11: Email Sender, Outbox, Outbound Worker

### Цель
Отправка email-ответов с threading. Гарантированная доставка через outbox.

### Файлы
- `internal/sender/sender.go`
- `internal/sender/templates.go`
- `internal/sender/sender_test.go`
- `internal/worker/outbound.go`

### Интерфейс
```go
type EmailSender struct {
    smtpClient *gomail.Dialer
    from       string
}

func (s *EmailSender) SendAcknowledgement(ctx context.Context, data *AcknowledgementData) error
func (s *EmailSender) SendCommentReply(ctx context.Context, data *CommentReplyData) error
func (s *EmailSender) SendStatusChange(ctx context.Context, data *StatusChangeData) error
```

### Outbound Worker
- Читает outbox из Store каждые 15 секунд
- Отправляет через EmailSender
- Retry с задержкой: 1м, 5м, 15м, 1ч
- После 5 попыток — помечает failed, алерт в лог

### Критерии приёмки
- Acknowledgement отправляется с корректным In-Reply-To
- CommentReply сохраняет цепочку (References)
- Outbox: письмо попадает в очередь → отправляется → помечается sent
- При ошибке SMTP — retry работает
- Тесты с mock SMTP

---

## Этап 12: Webhook Handler — приём событий из Plane

### Цель
Обработка событий из Plane (новый комментарий, изменение статуса).

### Файлы
- `internal/webhook/handler.go`
- `internal/webhook/validator.go`
- `internal/webhook/handler_test.go`

### Обрабатываемые события
- `issue.comment.created` — отправить комментарий автору письма
- `issue.updated` — отправить уведомление о смене статуса

### Защита от петель
- В начало каждого исходящего комментария добавлять маркер `[MAILBRIDGE-INTERNAL]`
- При получении webhook'а проверять наличие маркера → игнорировать

### Критерии приёмки
- Принимает webhook, валидирует подпись
- Игнорирует события от самого шлюза
- Находит оригинальное письмо по issue_id
- Вызывает EmailSender.SendCommentReply
- Тесты с httptest.Server и моками
```

---

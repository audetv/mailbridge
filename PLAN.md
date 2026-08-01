# Mailbridge — План разработки v2

## Статус

**Версия:** v0.18.1
**Выполненные этапы (v1):** 0–16 (зафиксированы в `PLAN.v1.md`)

### Что работает
- Приём писем через IMAP
- Классификация через rules.yml (NLP + стемминг)
- Создание задач в Plane (work-items, labels)
- Обработка ответов (добавление комментариев)
- Webhook'и от Plane
- Отказоустойчивость (реконнекты, health-проверки, метрики)
- Конфигурация через YAML

### Проблемы, выявленные в боевой эксплуатации
1. Plane не позволяет сменить проект задачи — ошибка классификации фатальна
2. Plane не даёт единого потока для руководителя
3. Plane free не имеет интеграции с GitHub
4. Классификация на rules.yml недостаточно точна (~60%)

### Решение
Отказ от Plane как базы задач. Разработка собственного helpdesk-модуля с веб-интерфейсом на Vue.js 3 и хранением задач в SQLite.

---

## Этап 17: Таблица задач в БД + REST API

### Цель
Спроектировать и реализовать таблицы для хранения задач, комментариев, вложений. Создать REST API для CRUD-операций.

### Файлы
```
internal/store/store.go            — новые модели: Task, TaskComment, TaskAttachment
internal/store/sqlite/sqlite.go    — миграции, CRUD-методы
internal/store/sqlite/sqlite_test.go
internal/web/api.go                — REST API handlers
internal/web/api_test.go
internal/web/auth.go               — базовая аутентификация (JWT)
internal/web/auth_test.go
cmd/mailbridge/main.go             — регистрация API-роутов
```

### Модели данных

```sql
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id TEXT NOT NULL UNIQUE,
    subject TEXT NOT NULL,
    body_text TEXT NOT NULL,
    body_html TEXT DEFAULT '',
    from_email TEXT NOT NULL,
    from_name TEXT DEFAULT '',
    project TEXT NOT NULL DEFAULT 'Входящие',
    type TEXT NOT NULL DEFAULT '',
    priority TEXT NOT NULL DEFAULT 'medium',
    status TEXT NOT NULL DEFAULT 'new',
    assignee TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE task_comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    author TEXT NOT NULL,
    body TEXT NOT NULL,
    direction TEXT NOT NULL DEFAULT 'in',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE task_attachments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size INTEGER NOT NULL,
    storage_path TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### API Endpoints

```
GET    /api/tasks                    — список задач (paginated, filterable)
GET    /api/tasks/:id                — задача с комментариями и вложениями
POST   /api/tasks/:id/reply          — ответ клиенту
PATCH  /api/tasks/:id                — обновить статус/проект/исполнителя/тип/приоритет
POST   /api/auth/login               — вход (логин/пароль → JWT)
GET    /api/auth/me                  — текущий пользователь
```

### Критерии приёмки
- [ ] Таблицы создаются миграцией
- [ ] API создаёт задачу из письма (процессор адаптирован)
- [ ] API возвращает список задач с фильтрацией
- [ ] API позволяет обновить статус/проект/исполнителя
- [ ] Аутентификация работает (JWT)
- [ ] `make test` проходит

---

## Этап 18: Интеграция процессора с новой БД

### Цель
Перенаправить создание задач из Plane API в собственную БД. Убрать зависимость от Plane для хранения задач.

### Файлы
```
internal/processor/processor.go     — переписать createNewIssue, addCommentToIssue
internal/processor/processor_test.go
cmd/mailbridge/main.go              — убрать инициализацию PlaneClient для процессора
```

### Логика
- `createNewIssue` — пишет задачу в таблицу `tasks` вместо Plane API
- `addCommentToIssue` — пишет комментарий в `task_comments` вместо Plane API
- `resolveProject` — определяет проект (строка-категория) из классификации
- `resolveLabels` — заменяется на прямое присвоение type/priority
- Plane API остаётся только для загрузки списка проектов (опционально)

### Критерии приёмки
- [ ] Письмо → задача в таблице tasks
- [ ] Ответ на письмо → комментарий в task_comments
- [ ] Вложения сохраняются в task_attachments
- [ ] `make test` проходит

---

## Этап 19: Веб-интерфейс на Vue.js 3

### Этап 19.1: Инициализация проекта и роутинг

**Цель:** Создать структуру Vue-проекта, настроить Vite, Vue Router, подключить PrimeVue. Базовая страница входа и пустой дашборд с проверкой аутентификации.

**Файлы:**
- `frontend/package.json`
- `frontend/vite.config.js`
- `frontend/index.html`
- `frontend/src/main.js`
- `frontend/src/App.vue`
- `frontend/src/router/index.js`
- `frontend/src/views/LoginView.vue`
- `frontend/src/views/DashboardView.vue`
- `frontend/src/stores/auth.js`
- `frontend/src/api/client.js`

**Критерии приёмки:**
- [ ] `npm run dev` запускает dev-сервер
- [ ] `/login` показывает форму входа
- [ ] Успешный вход → редирект на `/`
- [ ] `/` показывает пустой дашборд (заглушку)
- [ ] Без токена редиректит на `/login`
- [ ] `npm run build` собирает production-бандл

---

### Этап 19.2: Таблица задач с фильтрами

**Цель:** Компонент таблицы задач с сортировкой, фильтрацией и пагинацией. Данные загружаются из API.

**Файлы:**
- `frontend/src/components/TaskTable.vue`
- `frontend/src/components/FilterBar.vue`
- `frontend/src/components/StatusBadge.vue`
- `frontend/src/stores/tasks.js`
- `frontend/src/views/DashboardView.vue` (обновление)
- `internal/web/api.go` (дополнить эндпоинт `GET /api/tasks` — pagination, filters)

**API:**
```
GET /api/tasks?page=1&per_page=50&project=ТРК&status=new&assignee=Иванов&search=баннер
Response: { tasks: [...], total: 150, page: 1, per_page: 50 }
```

**Критерии приёмки:**
- [ ] Таблица отображает список задач
- [ ] Колонки: ID, Дата, От кого, Тема, Тип, Приоритет, Проект, Статус, Исполнитель
- [ ] Сортировка по клику на заголовок колонки
- [ ] Фильтры: проект, статус, исполнитель (выпадающие списки)
- [ ] Поиск по теме/тексту (поле ввода)
- [ ] Пагинация (если задач >50)
- [ ] `make test` проходит

---

### Этап 19.3: Карточка задачи

**Цель:** Модальное окно или страница с деталями задачи: описание, вложения, комментарии, действия.

**Файлы:**
- `frontend/src/components/TaskCard.vue`
- `frontend/src/components/CommentList.vue`
- `frontend/src/views/TaskDetailView.vue`
- `frontend/src/router/index.js` (обновление — маршрут `/tasks/:id`)

**API:**
```
GET /api/tasks/:id
Response: {
  task: { id, subject, body_text, body_html, from_email, from_name, project, type,
          priority, status, assignee, created_at },
  comments: [{ id, author, body, direction, created_at }],
  attachments: [{ id, filename, content_type, size, storage_path }]
}
```

**Критерии приёмки:**
- [ ] Клик по задаче в таблице → открывается карточка
- [ ] Вкладка «Описание»: текст письма, вложения (ссылки)
- [ ] Вкладка «Комментарии»: список комментариев с автором и датой
- [ ] Из карточки можно вернуться к таблице
- [ ] `make test` проходит

---

### Этап 19.4: Действия с задачей

**Цель:** Изменить статус, проект, исполнителя через UI. Ответить клиенту.

**Файлы:**
- `frontend/src/components/ReplyForm.vue`
- `frontend/src/components/TaskCard.vue` (обновление — кнопки действий)
- `frontend/src/stores/tasks.js` (обновление — actions)

**API:**
```
PATCH /api/tasks/:id
Body: { "project": "Отель" }
ИЛИ: { "status": "in_progress" }
ИЛИ: { "assignee": "Иванов" }
Response: { task: {...} }

POST /api/tasks/:id/reply
Body: { "body": "Текст ответа" }
Response: { comment: {...} }
```

**Критерии приёмки:**
- [ ] Выпадающий список «Проект» — меняет проект задачи
- [ ] Выпадающий список «Статус» — меняет статус
- [ ] Выпадающий список «Исполнитель» — назначает исполнителя
- [ ] Форма ответа: ввод текста → отправка → комментарий появляется в списке
- [ ] Ответ уходит клиенту на email
- [ ] Все изменения сохраняются в БД
- [ ] `make test` проходит

---

### Этап 19.5: Интеграция с Go (embed + production build)

**Цель:** Настроить production-сборку: Vite собирает статику, Go внедряет через `embed` и раздаёт. Один бинарник.

**Файлы:**
- `cmd/mailbridge/main.go` (обновление — раздача статики)
- `Makefile` (обновление — сборка фронтенда перед Go)
- `frontend/vite.config.js` (обновление — production-настройки)

**Критерии приёмки:**
- [ ] `make build` собирает фронтенд и бекенд
- [ ] Бинарник запускается и отдаёт SPA на `:8080`
- [ ] API работает на `/api/`
- [ ] SPA работает без dev-сервера
- [ ] `make test` проходит

## Этап 20: ИИ-классификатор (локальный Qwen 14B)

### Цель
Повысить точность классификации до >85% с помощью локальной LLM.

### Архитектура
```
Mailbridge → текст письма → HTTP POST → Ollama API (localhost:11434)
                                              ↓
                                         Qwen 2.5 14B
                                              ↓
                                         JSON: {project, type, priority, confidence}
                                              ↓
Mailbridge ← если confidence < 0.7 → fallback на rules.yml
           ← если Ollama недоступна → fallback на rules.yml
```

### Файлы
```
internal/classifier/ai_based.go     — клиент Ollama API
internal/classifier/classifier.go   — CompositeClassifier (ИИ → rules → triage)
internal/config/config.go           — настройки Ollama
configs/rules.yml                   — флаг ai_enabled
cmd/mailbridge/main.go              — инициализация
```

### Промпт
```
Ты — классификатор обращений в техподдержку.
Определи проект, тип и приоритет по тексту обращения.

Проекты: ТРК, Отель, Фитнес-клуб, Театр, Мебельный центр, Складской комплекс,
         Кафе, Ледовая арена, Корпоративные сайты, Входящие
Типы: bug, feature, support, access, seo, content
Приоритеты: urgent, high, medium, low

Ответь ТОЛЬКО валидным JSON без форматирования:
{"project":"...","type":"...","priority":"...","confidence":0.0-1.0}

Текст обращения: {text}
```

### Критерии приёмки
- [ ] Ollama установлена, модель загружена
- [ ] AI-классификатор возвращает корректный JSON
- [ ] Fallback на rules.yml при ошибке
- [ ] Точность >85% на 30 тестовых письмах
- [ ] Задержка <2 секунд на запрос

---

## Этап 21: Финальное тестирование и приёмка v1.0.0

### Цель
Проверить полный цикл работы системы, зафиксировать v1.0.0.

### Действия
- [ ] Прогнать 50 реальных писем через полный цикл (email → задача → ответ)
- [ ] Проверить точность классификации (ИИ + rules)
- [ ] Проверить веб-интерфейс (дашборд, фильтры, ответы)
- [ ] Проверить отказоустойчивость (обрыв IMAP, перезапуск)
- [ ] Нагрузочный тест (100+ писем)
- [ ] Финальный коммит, тег v1.0.0
- [ ] README с инструкцией по развёртыванию

---

## Оценка трудозатрат

| Этап | Содержание | Часов |
|------|-----------|-------|
| 17 | БД + REST API | 12 |
| 18 | Интеграция процессора | 6 |
| 19 | Vue.js интерфейс | 34 |
| 20 | ИИ-классификатор | 8 |
| 21 | Тестирование и приёмка | 10 |
| **Итого** | | **70 часов** |
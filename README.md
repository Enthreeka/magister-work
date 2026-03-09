# codegen

Инструмент кодогенерации для Go-приложений на основе декларативного YAML-файла с требованиями.

По одному файлу `system-gen.yaml` генерирует все слои **Clean Architecture**: domain, repository, service, handler — готовые к использованию, типобезопасные, структурированные по принципу вертикального разбиения.

---

## Содержание

- [Установка](#установка)
- [Быстрый старт](#быстрый-старт)
- [Файл требований](#файл-требований-system-genyaml)
- [Команды CLI](#команды-cli)
- [Генерируемые слои](#генерируемые-слои)
- [Стратегии репозитория](#стратегии-репозитория)
- [Защита файлов и обратная совместимость](#защита-файлов-и-обратная-совместимость)
- [Архитектура проекта](#архитектура-проекта)
- [AI-провайдер (задел)](#ai-провайдер-задел)
- [Roadmap](#roadmap)

---

## Установка

```bash
git clone https://github.com/Enthreeka/magister-work
cd magister-work
go build -o codegen ./cmd/codegen
```

После сборки переместите бинарник в `$PATH`:

```bash
mv codegen /usr/local/bin/codegen
```

---

## Быстрый старт

**1. Создайте файл требований:**

```yaml
# system-gen.yaml
version: "1"
module: github.com/example/app
domain: user

transport:
  framework: gin
  method: POST
  url: /api/v1/users

input:
  - name: name
    type: string
    source: body
    required: true
  - name: email
    type: string
    source: body
    required: true

output:
  - name: id
    type: uuid
  - name: created_at
    type: time.Time

repository:
  strategy: native
  driver: pgx
  table: users
  schema: public
  operation: insert
  fields: [name, email]

generate:
  layers:
    - domain
    - repository
    - service
    - handler
  output_dir: internal/{{.Domain}}
```

**2. Проверьте схему:**

```bash
codegen validate
```

**3. Сгенерируйте код:**

```bash
codegen generate
```

**Результат:**

```
internal/user/
└── gen/
    ├── domain.gen.go      # интерфейсы, типы, ошибки
    ├── repository.gen.go  # реализация слоя хранения
    ├── service.gen.go     # бизнес-логика (TODO-stub)
    └── handler.gen.go     # Gin HTTP-обработчик
```

---

## Файл требований `system-gen.yaml`

### Корневые поля

| Поле | Тип | Описание |
|------|-----|----------|
| `version` | string | Версия формата схемы (сейчас `"1"`) |
| `module` | string | Go-модуль целевого проекта |
| `domain` | string | Имя функционального модуля (`user`, `order`, `product`) |

### `transport`

Описывает HTTP-эндпоинт.

| Поле | Значения | Описание |
|------|----------|----------|
| `framework` | `gin` | Веб-фреймворк (сейчас только gin) |
| `method` | `GET`, `POST`, `PUT`, `DELETE`, `PATCH` | HTTP-метод |
| `url` | string | Путь эндпоинта, например `/api/v1/users` |

### `input` и `output`

Списки полей запроса и ответа.

| Поле | Обязательное | Описание |
|------|-------------|----------|
| `name` | да | Имя поля (snake_case) |
| `type` | да | Тип данных (см. таблицу типов) |
| `source` | нет | Только для `input`: `body`, `query`, `header`, `path` |
| `required` | нет | Обязательность поля в запросе |

**Поддерживаемые типы:**

| YAML-тип | Go-тип |
|----------|--------|
| `string` | `string` |
| `int`, `int32`, `int64` | `int64`, `int32`, `int64` |
| `float`, `float32`, `float64` | `float64`, `float32`, `float64` |
| `bool` | `bool` |
| `uuid` | `string` |
| `time`, `time.Time` | `time.Time` |
| `[]byte` | `[]byte` |

### `repository`

| Поле | Значения | Описание |
|------|----------|----------|
| `strategy` | `native`, `sqlc` | Стратегия генерации репозитория |
| `driver` | `pgx`, `sqlx` | Драйвер БД (только для `native`) |
| `table` | string | Имя таблицы |
| `schema` | string | Схема БД (например `public`) |
| `operation` | `insert`, `select`, `update`, `delete` | SQL-операция |
| `fields` | `[]string` | Поля, задействованные в SQL-запросе |
| `sqlc.mode` | `generate`, `existing` | Режим sqlc (только для `sqlc`) |
| `sqlc.config` | string | Путь к `sqlc.yaml` (только для режима `existing`) |

### `generate`

| Поле | Описание |
|------|----------|
| `layers` | Список слоёв: `domain`, `repository`, `service`, `handler` |
| `output_dir` | Директория вывода. Поддерживает `{{.Domain}}` |

### `service` (опционально)

```yaml
service:
  description: >
    Создать пользователя. Проверить уникальность email.
  feature_flags:
    - name: send_welcome_email
      description: Отправить письмо после создания
      default: false
```

Используется AI-провайдером для генерации бизнес-логики (см. [AI-провайдер](#ai-провайдер-задел)).

---

## Команды CLI

### `codegen generate`

Генерирует код по схеме.

```bash
codegen generate [flags]
```

| Флаг | Сокращение | По умолчанию | Описание |
|------|-----------|-------------|----------|
| `--schema` | `-s` | `system-gen.yaml` | Путь к файлу требований |
| `--output` | `-o` | из схемы | Переопределить директорию вывода |
| `--layers` | `-l` | все | Генерировать только указанные слои (через запятую) |
| `--dry-run` | | `false` | Показать файлы без записи |
| `--force` | | `false` | Перезаписать пользовательские (не-generated) файлы |
| `--force-breaking` | | `false` | Разрешить breaking changes |

**Примеры:**

```bash
# Генерация только domain и service
codegen generate --layers domain,service

# Предпросмотр без записи
codegen generate --dry-run

# Другой файл требований
codegen generate --schema ./user/system-gen.yaml --output ./internal/user
```

### `codegen validate`

Проверяет корректность схемы без генерации файлов.

```bash
codegen validate --schema system-gen.yaml
```

### `codegen diff`

Показывает изменения схемы относительно последней генерации (читает `.codegen.lock`).

```bash
codegen diff
```

Пример вывода:
```
2 change(s) detected:
  [ERROR] transport.method: changed from "POST" to "GET"
  [WARNING] output[created_at]: output field removed

This schema has breaking changes. Use --force-breaking with generate to proceed.
```

### `codegen version`

```bash
codegen version
# codegen v0.1.0
```

---

## Генерируемые слои

### `domain` — контракты

Файл `gen/domain.gen.go` содержит:

- **Типы запроса и ответа** (`UserRequest`, `UserResponse`)
- **Интерфейс репозитория** (`UserRepository`) — контракт для слоя хранения
- **Интерфейс сервиса** (`UserServiceIface`) — контракт для бизнес-логики
- **Доменные ошибки** (`ErrUserNotFound`, `ErrUserValidation`, `ErrUserDomain`, `ErrUserInternal`)

```go
// Использование в основном приложении
var _ domain.UserRepository   = (*repository.UserRepositoryImpl)(nil) // проверка на этапе компиляции
var _ domain.UserServiceIface = (*service.UserService)(nil)
```

### `repository` — слой хранения

Файл `gen/repository.gen.go` содержит:

- Struct с зависимостью от `*pgxpool.Pool` (или `*sqlx.DB`)
- Конструктор `NewUserRepository`
- Реализацию SQL-операции с типизированным `Scan`

### `service` — бизнес-логика

Файл `gen/service.gen.go` содержит готовую структуру сервиса с TODO-stub'ом метода. Пользователь заполняет логику:

```go
func (s *UserService) Create(ctx context.Context, req *domain.UserRequest) (*domain.UserResponse, error) {
    // TODO: implement Create
    // Generated by codegen (noop provider). Replace with your logic.
    panic("not implemented: Create")
}
```

### `handler` — HTTP-обработчик (Gin)

Файл `gen/handler.gen.go` содержит:

- Struct с зависимостью от `domain.UserServiceIface`
- Метод `Register(r *gin.RouterGroup)` для монтирования маршрутов
- Автоматическое связывание (`ShouldBindJSON`, `ShouldBindQuery`, `c.GetHeader`, `c.Param`) по полю `source`
- Маппинг доменных ошибок на HTTP-коды (404, 400, 409, 500)

---

## Стратегии репозитория

### `strategy: native`

Codegen самостоятельно генерирует SQL-запросы и Go-код слоя репозитория.

```yaml
repository:
  strategy: native
  driver: pgx       # pgx | sqlx
  table: users
  schema: public
  operation: insert
  fields: [name, email]
```

### `strategy: sqlc`

#### Режим `generate`

Codegen создаёт `sqlc.yaml` и `.sql`-файл запросов, затем вызывает `sqlc generate`.

> Требует установленного `sqlc`: `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`

```yaml
repository:
  strategy: sqlc
  table: users
  schema: public
  operation: insert
  fields: [name, email]
  sqlc:
    mode: generate
```

Codegen создаёт:
```
internal/user/gen/sqlc/
├── sqlc.gen.yaml       # DO NOT EDIT
└── queries.gen.sql     # DO NOT EDIT
internal/user/db/       # ← sqlc output
├── db.go
├── models.go
└── query.sql.go
```

#### Режим `existing`

Пользователь уже использует sqlc. Codegen читает существующий `sqlc.yaml`, определяет интерфейс `Querier` и генерирует только `service` и `handler` поверх него.

```yaml
repository:
  strategy: sqlc
  sqlc:
    mode: existing
    config: ./sqlc.yaml
```

---

## Защита файлов и обратная совместимость

### Защита сгенерированных файлов

Все файлы в директории `gen/` содержат заголовок:

```go
// Code generated by codegen DO NOT EDIT.
// source: system-gen.yaml
// version: 0.1.0
```

При перегенерации codegen:
- **Перезаписывает** файлы с этим заголовком
- **Пропускает** файлы без заголовка (пользовательский код)
- **Предупреждает** о пропущенных файлах

Чтобы принудительно перезаписать пользовательские файлы: `--force`.

### `.codegen.lock`

После каждой успешной генерации создаётся (или обновляется) файл `.codegen.lock` — снимок схемы домена. При следующем запуске `generate` или `diff` codegen сравнивает текущую схему с сохранённой.

**Breaking changes (блокируют генерацию):**
- Изменение HTTP-метода
- Изменение типа существующего поля
- Удаление обязательного поля запроса
- Изменение операции репозитория

**Non-breaking changes (предупреждение):**
- Изменение URL
- Удаление поля ответа
- Добавление новых полей

Для применения breaking changes: `codegen generate --force-breaking`.

---

## Архитектура проекта

```
.
├── cmd/codegen/             # точка входа CLI (cobra)
│   ├── main.go
│   ├── generate.go
│   ├── validate.go
│   └── diff.go
│
├── internal/
│   ├── schema/              # парсинг и валидация system-gen.yaml
│   │   ├── types.go
│   │   ├── parser.go
│   │   └── validator.go
│   │
│   ├── generator/           # ядро — не знает о языке генерации
│   │   ├── engine.go        # Engine, LayerGenerator interface, TemplateData
│   │   ├── file.go          # File struct, DO NOT EDIT detection
│   │   ├── compat/
│   │   │   └── checker.go   # .codegen.lock + breaking change detector
│   │   ├── ai/
│   │   │   ├── provider.go  # BusinessLogicProvider interface
│   │   │   ├── noop.go      # NoopProvider (TODO stub)
│   │   │   └── registry.go
│   │   └── repository/
│   │       ├── strategy.go  # Strategy interface
│   │       ├── native.go    # NativeStrategy (pgx/sqlx)
│   │       └── sqlc.go      # SqlcStrategy (generate/existing)
│   │
│   └── golang/              # Go language plugin
│       ├── generator.go     # NewEngine + BuildTemplateData
│       ├── templates.go
│       ├── tmplsrc/         # embedded шаблоны (leaf-пакет)
│       │   ├── domain.go.tmpl
│       │   ├── service.go.tmpl
│       │   └── handler_gin.go.tmpl
│       └── layers/
│           ├── helpers.go   # toCamel, toSnake, toGoType, templateFuncs
│           ├── domain.go
│           ├── repository.go
│           ├── service.go
│           └── handler.go
│
├── pkg/
│   └── typemap/
│       └── typemap.go       # YAML тип → Go тип
│
├── testdata/
│   └── schemas/
│       └── user_create.yaml # пример схемы
│
└── .codegen.lock            # автосоздаётся после первой генерации
```

### Принципы дизайна

- **Ядро не знает о языке** — `internal/generator` работает через интерфейсы `LayerGenerator` и `Strategy`. Go-специфика изолирована в `internal/golang`.
- **Шаблоны встроены в бинарник** — через `//go:embed`, никаких внешних файлов при деплое.
- **Защита пользовательского кода** — перезаписываются только файлы с DO NOT EDIT заголовком.
- **Интерфейсы на стороне потребителя** — `UserRepository` и `UserServiceIface` определены в `domain`, а не в пакетах-реализациях.

---

## AI-провайдер (задел)

Интерфейс `BusinessLogicProvider` готов к подключению AI-агентов для генерации тела методов бизнес-логики.

Сейчас работает `NoopProvider` (TODO-stub). В будущем:

```yaml
# system-gen.yaml
ai_provider:
  name: anthropic       # anthropic | openai | ollama | noop
  model: claude-opus-4-6

service:
  description: >
    Создать пользователя. Проверить уникальность email.
    Вернуть DomainError если пользователь уже существует.
  feature_flags:
    - name: send_welcome_email
      description: Отправить приветственное письмо после регистрации
      default: false
```

Подключение нового провайдера:

```go
// Реализуй интерфейс
type AnthropicProvider struct { ... }
func (p AnthropicProvider) Name() string { return "anthropic" }
func (p AnthropicProvider) GenerateMethodBody(ctx context.Context, req ai.MethodRequest) (string, error) { ... }

// Зарегистрируй
ai.Register(AnthropicProvider{})
```

---

## Roadmap

- [x] Go: domain, repository (native/sqlc), service, handler (Gin)
- [x] CLI: generate, validate, diff, version
- [x] Compat checker + `.codegen.lock`
- [x] Защита пользовательских файлов (DO NOT EDIT)
- [x] sqlc интеграция (generate + existing)
- [x] AI-провайдер интерфейс + NoopProvider
- [ ] Fiber и Echo handler templates
- [ ] Python language plugin
- [ ] Rust language plugin
- [ ] Anthropic / OpenAI AI-провайдеры
- [ ] Golden file тесты
- [ ] `codegen init` — интерактивный wizard для создания схемы

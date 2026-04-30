# protodefault

`protodefault` — это Go-библиотека для применения значений по умолчанию к protobuf-сообщениям на основе кастомных опций в `.proto`-файлах.

## Установка

```bash
go get github.com/yurasolovjov/protodefault
```

## Быстрый старт

1. Объявите опцию в вашем `.proto`:

```proto
syntax = "proto3";
package myapp.v1;

import "defaults/v1/options.proto";

message Config {
  string host = 1 [(defaults.v1.default_value) = "localhost"];
  int32 port = 2 [(defaults.v1.default_value) = "8080"];
}
```

2. Примените дефолты в коде:

```go
package main

import (
    "github.com/yurasolovjov/protodefault"
    myappv1 "yourproject/gen/myapp/v1"
)

func main() {
    cfg := &myappv1.Config{}
    if err := protodefault.Apply(cfg); err != nil {
        panic(err)
    }
    // cfg.Host == "localhost"
    // cfg.Port == 8080
}
```

## Принцип работы

Библиотека работает полностью в **рантайме**, без кодогенерации. Она использует механизм **Protobuf Reflection** — API для динамической работы с protobuf-сообщениями, предоставляемый пакетом `google.golang.org/protobuf/reflect/protoreflect`.

### Зачем нужен Protobuf Reflection (`protoreflect`)

В обычном Go-коде вы работаете с protobuf-сообщениями как с обычными структурами: обращаетесь к полям по имени, присваиваете значения напрямую. Но что если вам нужно написать **универсальную** библиотеку, которая работает с **любым** protobuf-сообщением, не зная его типа заранее?

Именно для этого существует `protoreflect`. Это аналог стандартного пакета `reflect` в Go, но специально для protobuf. Он позволяет:

1. **Получить метаданные о схеме** (дескрипторы): какие поля есть в сообщении, их типы, номера, опции.
2. **Читать и записывать значения полей** динамически, без знания конкретного типа на этапе компиляции.
3. **Обходить вложенные структуры** рекурсивно.

#### Ключевые концепции `protoreflect`

**Дескрипторы (Descriptors)** — это метаданные о схеме protobuf. Они описывают структуру сообщений, но не содержат данных:

```go
// Получаем дескриптор сообщения
md := msg.ProtoReflect().Descriptor()

// Получаем список всех полей
fields := md.Fields()

// Для каждого поля можно узнать:
for i := 0; i < fields.Len(); i++ {
    fd := fields.Get(i)
    fd.Name()       // имя поля
    fd.Number()     // номер поля в proto
    fd.Kind()       // тип: string, int32, message, enum и т.д.
    fd.IsList()     // это repeated поле?
    fd.IsMap()      // это map поле?
    fd.Options()    // опции поля (включая наши кастомные)
}
```

**Интерфейс `protoreflect.Message`** — это обёртка над конкретным экземпляром сообщения, предоставляющая динамический доступ к данным:

```go
// Получаем reflection-интерфейс
rm := msg.ProtoReflect()

// Проверяем, установлено ли значение поля
if rm.Has(fieldDescriptor) {
    // поле было явно задано
}

// Читаем значение
value := rm.Get(fieldDescriptor)

// Записываем значение
rm.Set(fieldDescriptor, protoreflect.ValueOfString("hello"))

// Создаём новый экземпляр вложенного сообщения
newMsg := rm.NewField(fieldDescriptor)
```

### Как работает механизм `extend google.protobuf.FieldOptions`

Protobuf имеет встроенную систему **опций** — метаданных, которые можно прикреплять к различным элементам схемы (файлам, сообщениям, полям, сервисам и т.д.). Стандартные опции определены в `google/protobuf/descriptor.proto`.

Механизм `extend` позволяет **расширять** эти стандартные опции своими собственными полями. Вот как это работает:

#### 1. Объявление расширения

В файле `options.proto` мы объявляем новое поле для `FieldOptions`:

```proto
syntax = "proto3";
package defaults.v1;

import "google/protobuf/descriptor.proto";

extend google.protobuf.FieldOptions {
  string default_value = 50001;  // номер из приватного диапазона
}
```

**Что здесь происходит:**
- `google.protobuf.FieldOptions` — это стандартное сообщение protobuf, содержащее опции для полей.
- `extend` добавляет к нему новое поле `default_value` типа `string`.
- Номер `50001` выбран из диапазона 50000-99999, зарезервированного для приватных расширений.

#### 2. Использование опции в `.proto` файлах

Теперь в любом `.proto` файле можно использовать эту опцию:

```proto
import "defaults/v1/options.proto";

message Config {
  string host = 1 [(defaults.v1.default_value) = "localhost"];
  int32 port = 2 [(defaults.v1.default_value) = "8080"];
}
```

Синтаксис `[(package.option_name) = value]` — это способ задать значение расширения.

#### 3. Что делает компилятор `protoc`

Когда `protoc` компилирует `.proto` файл:

1. Он видит опцию `(defaults.v1.default_value)` на поле.
2. Записывает значение опции в **бинарный дескриптор** сообщения.
3. Этот дескриптор встраивается в сгенерированный Go-код.

В результате, когда ваша программа запускается, информация о дефолтных значениях уже "зашита" в скомпилированный код и доступна через reflection.

#### 4. Чтение опции в рантайме

Библиотека `protodefault` извлекает значение опции так:

```go
import (
    "google.golang.org/protobuf/proto"
    defaultsv1 "github.com/yurasolovjov/protodefault/proto/defaults/v1"
)

// fd — это FieldDescriptor поля
options := fd.Options()  // получаем FieldOptions

// Проверяем, есть ли наше расширение
if proto.HasExtension(options, defaultsv1.E_DefaultValue) {
    // Извлекаем значение
    defaultStr := proto.GetExtension(options, defaultsv1.E_DefaultValue).(string)
    // defaultStr содержит "localhost" или "8080"
}
```

`defaultsv1.E_DefaultValue` — это переменная типа `protoreflect.ExtensionType`, автоматически сгенерированная из нашего `options.proto`. Она служит "ключом" для доступа к расширению.

### Алгоритм работы `Apply()`

Когда вы вызываете `protodefault.Apply(msg)`, библиотека выполняет следующие шаги:

```
Apply(msg)
│
├─► Получить reflection-интерфейс: rm = msg.ProtoReflect()
│
├─► Получить дескриптор: md = rm.Descriptor()
│
├─► Для каждого поля в md.Fields():
│   │
│   ├─► Извлечь опцию default_value (если есть)
│   │
│   ├─► Проверить, нужно ли применять дефолт:
│   │   ├─ Для message/optional/oneof: rm.Has(fd) == false
│   │   ├─ Для proto3 scalar: значение == zero-value
│   │   └─ Для repeated/map: len == 0
│   │
│   ├─► Если нужно — распарсить строку в нужный тип и установить
│   │
│   └─► Если поле — message, рекурсивно вызвать Apply()
│
└─► Вернуть nil или ошибку
```

### Почему Reflection, а не кодогенерация?

Альтернативный подход — написать плагин для `protoc`, который генерирует Go-код с методами `ApplyDefaults()` для каждого сообщения. Мы выбрали reflection по следующим причинам:

| Критерий | Reflection | Кодогенерация |
|----------|------------|---------------|
| Простота интеграции | Одна зависимость, работает сразу | Нужно настраивать плагин в build pipeline |
| Размер бинарника | Минимальный overhead | Дополнительный код для каждого сообщения |
| Производительность | Медленнее (reflection) | Быстрее (прямой доступ к полям) |
| Гибкость | Работает с любыми сообщениями | Только с теми, для которых сгенерирован код |

Для типичного use-case (применение дефолтов при загрузке конфига) производительность reflection более чем достаточна, а простота интеграции — критически важна.

## Семантика и ограничения

### Presence (Наличие значения)

Это ключевой аспект работы библиотеки. В зависимости от типа поля, логика "нужно ли применять дефолт" меняется:

- **Messages / Oneof / Optional (proto3)**: Эти поля обладают поддержкой явного присутствия. Мы вызываем `Has()`. Если оно возвращает `false`, дефолт применяется.
- **Proto3 Scalars (без optional)**: В proto3 обычные числа, строки и булевы значения не имеют состояния "не задано" на уровне протокола. Они всегда имеют значение (даже если это `0` или `""`).
    - **Ограничение**: Для таких полей дефолт применяется, только если текущее значение равно **zero-value**. Это означает, что если вы явно установите `0` в поле, где дефолт `42`, библиотека перезапишет его на `42`. Для решения этой проблемы используйте ключевое слово `optional`.
- **Repeated / Map**: Дефолт применяется, только если коллекция пуста (`len == 0`). Мы не объединяем дефолтные значения с пользовательскими.

### Oneof

Логика для `oneof` спроектирована так, чтобы избежать неоднозначности:
1. Если какое-то поле в `oneof` уже установлено пользователем, библиотека ничего не меняет (но может зайти внутрь, если это `message`).
2. Если `oneof` пуст, библиотека ищет поля с `default_value`.
3. Если найдено **ровно одно** такое поле — оно инициализируется.
4. Если найдено **несколько** — это считается ошибкой конфигурации, так как библиотека не может выбрать между ними.

### Рекурсия и защита от циклов

Библиотека выполняет глубокий обход:
- Она заходит во вложенные сообщения, даже если они не были инициализированы (но только если в их схеме есть дефолты).
- Чтобы избежать `stack overflow` на циклических схемах (например, `A` содержит `B`, а `B` содержит `A`), мы используем `visited-set` на основе полных имен дескрипторов (`FullName`) при предварительном анализе схемы.

## Поддерживаемые типы и форматы

| Категория | Тип | Формат default_value |
|---|---|---|
| Скаляры | `bool`, `int32/64`, `string`, `bytes`, и др. | Текстовое представление (`"true"`, `"42"`, `"hello"`) |
| Enum | `enum` | Имя константы (`"LOG_LEVEL_INFO"`) или номер (`"2"`) |
| Repeated | `repeated T` | JSON-массив (`'["a", "b"]'`) |
| Map | `map<K, V>` | JSON-объект (`'{"key": "value"}'`) |
| WKT | `google.protobuf.Duration` | Формат `time.ParseDuration` (`"15s"`, `"1h"`) |
| WKT | `google.protobuf.Timestamp` | RFC3339 (`"2026-01-01T00:00:00Z"`) |
| WKT | `google.protobuf.Struct` | JSON-объект |
| WKT | `google.protobuf.ListValue` | JSON-массив |
| WKT | `google.protobuf.Value` | Любой JSON |
| WKT | `google.protobuf.FieldMask` | Строка через запятую (`"a.b,c"`) |
| WKT | `google.protobuf.*Value` (Wrappers) | Как у соответствующего скаляра |

## Генерация кода

Для использования кастомной опции вам понадобятся сгенерированные файлы. Рекомендуется использовать `buf` или `protoc`:

```bash
protoc -I proto --go_out=. --go_opt=module=github.com/yurasolovjov/protodefault proto/defaults/v1/options.proto
```

## Как устроен Buf

[Buf](https://buf.build/) — это современный инструментарий для работы с Protocol Buffers, созданный как замена традиционному `protoc`. Buf решает множество проблем, с которыми сталкиваются разработчики при использовании protobuf: сложность настройки `protoc`, управление зависимостями, линтинг и форматирование схем.

### Архитектура Buf

Buf состоит из нескольких компонентов:

```
┌─────────────────────────────────────────────────────────────────┐
│                         Buf CLI                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │  buf build  │  │ buf generate│  │  buf lint   │              │
│  │             │  │             │  │             │              │
│  │ Компиляция  │  │ Генерация   │  │ Проверка    │              │
│  │ .proto в    │  │ кода через  │  │ стиля и     │              │
│  │ FileDescr.  │  │ плагины     │  │ best pract. │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
│                                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │ buf breaking│  │  buf format │  │  buf dep    │              │
│  │             │  │             │  │             │              │
│  │ Проверка    │  │ Авто-       │  │ Управление  │              │
│  │ обратной    │  │ форматиро-  │  │ зависимос-  │              │
│  │ совмест.    │  │ вание       │  │ тями        │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
│                                                                  │
├─────────────────────────────────────────────────────────────────┤
│                    Buf Schema Registry (BSR)                     │
│                                                                  │
│  Централизованный реестр proto-схем, аналог npm/Maven для proto │
│  URL: https://buf.build/                                         │
└─────────────────────────────────────────────────────────────────┘
```

### Ключевые файлы конфигурации

#### `buf.yaml` — конфигурация модуля

Определяет proto-модуль и его зависимости:

```yaml
version: v2
modules:
  - path: proto                    # Директория с .proto файлами
deps:
  - buf.build/bufbuild/protovalidate  # Зависимость из BSR
lint:
  use:
    - STANDARD                     # Набор правил линтинга
breaking:
  use:
    - FILE                         # Правила проверки совместимости
```

#### `buf.gen.yaml` — конфигурация генерации кода

Определяет, какие плагины использовать и куда класть результат:

```yaml
version: v2
managed:
  enabled: true
  override:
    - file_option: go_package_prefix
      value: github.com/myorg/myproject/gen
plugins:
  - remote: buf.build/protocolbuffers/go  # Плагин для Go
    out: gen
    opt:
      - paths=source_relative
  - remote: buf.build/grpc/go             # gRPC плагин
    out: gen
    opt:
      - paths=source_relative
```

#### `buf.lock` — lock-файл зависимостей

Автоматически генерируется командой `buf dep update`. Фиксирует точные версии (коммиты) всех зависимостей:

```yaml
version: v2
deps:
  - name: buf.build/bufbuild/protovalidate
    commit: a6c49f84cc0f4e5b69b6b7b3c0f3d8e1
    digest: shake256:abc123...
```

### Как Buf разрешает зависимости

Когда вы пишете в `.proto` файле:

```proto
import "buf/validate/validate.proto";
```

Buf выполняет следующие шаги:

1. **Читает `buf.yaml`** и находит зависимость `buf.build/bufbuild/protovalidate`.

2. **Проверяет `buf.lock`** на наличие закэшированной версии.

3. **Скачивает модуль из BSR** (если не закэширован) в локальный кэш:
   - macOS/Linux: `~/.cache/buf/`
   - Windows: `%LOCALAPPDATA%\buf\`

4. **Разрешает импорт** `buf/validate/validate.proto` относительно скачанного модуля.

### Сравнение Buf и protoc

| Аспект | protoc | Buf |
|--------|--------|-----|
| Управление зависимостями | Ручное (копировать файлы или `-I` пути) | Автоматическое через BSR |
| Конфигурация | Длинные командные строки | Декларативные YAML-файлы |
| Линтинг | Нет встроенного | `buf lint` с настраиваемыми правилами |
| Проверка совместимости | Нет | `buf breaking` |
| Форматирование | Нет | `buf format` |
| Удалённые плагины | Нужно устанавливать локально | Запуск в облаке через BSR |
| Кэширование | Нет | Встроенное |

### Пример рабочего процесса с Buf

```bash
# 1. Инициализация модуля
buf config init

# 2. Добавление зависимости (редактируем buf.yaml)
# deps:
#   - buf.build/bufbuild/protovalidate

# 3. Обновление lock-файла
buf dep update

# 4. Проверка синтаксиса и стиля
buf lint

# 5. Генерация кода
buf generate

# 6. Проверка обратной совместимости (в CI)
buf breaking --against '.git#branch=main'
```

## Где находится `buf/validate/validate.proto`

Файл `buf/validate/validate.proto` — это основной файл библиотеки [protovalidate](https://github.com/bufbuild/protovalidate), которая предоставляет декларативные правила валидации для protobuf-сообщений.

### Расположение в Buf Schema Registry (BSR)

Официальный модуль размещён в BSR по адресу:

```
buf.build/bufbuild/protovalidate
```

**Веб-интерфейс для просмотра:** https://buf.build/bufbuild/protovalidate/docs

### Структура модуля protovalidate

```
buf.build/bufbuild/protovalidate/
├── buf/
│   └── validate/
│       ├── validate.proto      # Основные правила валидации
│       ├── expression.proto    # CEL-выражения для кастомных правил
│       └── priv/
│           └── private.proto   # Внутренние типы (не для публичного использования)
└── buf.yaml                    # Конфигурация модуля
```

### Как подключить protovalidate

#### Способ 1: Через Buf (рекомендуется)

1. Добавьте зависимость в `buf.yaml`:

```yaml
version: v2
deps:
  - buf.build/bufbuild/protovalidate
```

2. Обновите lock-файл:

```bash
buf dep update
```

3. Используйте в `.proto`:

```proto
syntax = "proto3";

import "buf/validate/validate.proto";

message User {
  string email = 1 [(buf.validate.field).string.email = true];
  int32 age = 2 [(buf.validate.field).int32 = { gte: 0, lte: 150 }];
}
```

#### Способ 2: Через protoc (ручное управление)

Если вы используете `protoc` напрямую, нужно скачать proto-файлы вручную:

```bash
# Клонируем репозиторий
git clone https://github.com/bufbuild/protovalidate.git

# Структура proto-файлов
ls protovalidate/proto/protovalidate/
# buf/validate/validate.proto
# buf/validate/expression.proto
# buf/validate/priv/private.proto

# Компиляция с указанием пути
protoc \
  -I protovalidate/proto/protovalidate \
  -I your_proto_dir \
  --go_out=. \
  your_proto_dir/your_file.proto
```

### Содержимое `validate.proto`

Файл `buf/validate/validate.proto` определяет:

1. **Расширение `FieldOptions`** для добавления правил валидации к полям:

```proto
extend google.protobuf.FieldOptions {
  optional FieldConstraints field = 1159;
}
```

2. **Сообщение `FieldConstraints`** с правилами для разных типов:

```proto
message FieldConstraints {
  // Общие ограничения
  repeated Constraint cel = 23;           // CEL-выражения
  bool required = 25;                      // Обязательное поле
  bool ignore = 27;                        // Игнорировать при валидации

  // Типо-специфичные ограничения (oneof)
  oneof type {
    FloatRules float = 1;
    DoubleRules double = 2;
    Int32Rules int32 = 3;
    Int64Rules int64 = 4;
    // ... и так далее для всех типов
    StringRules string = 14;
    BytesRules bytes = 15;
    EnumRules enum = 16;
    RepeatedRules repeated = 18;
    MapRules map = 19;
  }
}
```

3. **Правила для каждого типа**, например `StringRules`:

```proto
message StringRules {
  optional string const = 1;        // Точное значение
  optional uint64 len = 19;         // Точная длина
  optional uint64 min_len = 2;      // Минимальная длина
  optional uint64 max_len = 3;      // Максимальная длина
  optional string pattern = 6;      // Регулярное выражение
  optional string prefix = 7;       // Префикс
  optional string suffix = 8;       // Суффикс
  optional string contains = 9;     // Содержит подстроку
  repeated string in = 10;          // Одно из значений
  repeated string not_in = 11;      // Не одно из значений

  // Well-known форматы
  oneof well_known {
    bool email = 12;
    bool hostname = 13;
    bool ip = 14;
    bool uri = 17;
    bool uuid = 22;
    // ... и другие
  }
}
```

### Интеграция protodefault и protovalidate

Эти две библиотеки отлично дополняют друг друга:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Порядок обработки                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. Парсинг         YAML/JSON → proto.Message                   │
│     ─────────────────────────────────────────────────────────   │
│                              │                                   │
│                              ▼                                   │
│  2. protodefault    Заполнение пустых полей дефолтами           │
│     ─────────────────────────────────────────────────────────   │
│                              │                                   │
│                              ▼                                   │
│  3. protovalidate   Проверка всех ограничений                   │
│     ─────────────────────────────────────────────────────────   │
│                              │                                   │
│                              ▼                                   │
│  4. Использование   Конфиг готов к работе                       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

Пример комбинированного использования:

```proto
message ServerConfig {
  // Обязательное поле без дефолта — валидация упадёт, если не задано
  string host = 1 [(buf.validate.field).string.min_len = 1];

  // Опциональное поле с дефолтом — всегда пройдёт валидацию
  int32 port = 2 [
    (defaults.v1.default_value) = "8080",
    (buf.validate.field).int32 = { gte: 1, lte: 65535 }
  ];

  // Опциональное поле с дефолтом и форматом
  string log_level = 3 [
    (defaults.v1.default_value) = "info",
    (buf.validate.field).string = { in: ["debug", "info", "warn", "error"] }
  ];
}
```

```go
package main

import (
    "github.com/yurasolovjov/protodefault"
    "buf.build/go/protovalidate"
)

func main() {
    cfg := &ServerConfig{Host: "localhost"}

    // 1. Применяем дефолты
    if err := protodefault.Apply(cfg); err != nil {
        panic(err)
    }
    // cfg.Port == 8080, cfg.LogLevel == "info"

    // 2. Валидируем
    validator, _ := protovalidate.New()
    if err := validator.Validate(cfg); err != nil {
        panic(err)
    }

    // Конфиг готов к использованию
}
```

Смотрите полный пример в [example/with-validation/](example/with-validation/).
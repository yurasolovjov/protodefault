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
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
import "github.com/yurasolovjov/protodefault"

func main() {
    cfg := &myappv1.Config{}
    protodefault.MustApply(cfg)
    // cfg.Host == "localhost"
    // cfg.Port == 8080
}
```

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

## Семантика и ограничения

### Presence (Наличие значения)
- **Messages / Oneof / Optional**: Используется `Has()`. Дефолт применяется, если поле не задано.
- **Proto3 Scalars (без optional)**: Дефолт применяется, если текущее значение равно **zero-value** (0, "", false). Это связано с тем, что в proto3 невозможно отличить явно установленный 0 от отсутствующего значения для обычных скаляров.
- **Repeated / Map**: Дефолт применяется, только если коллекция пуста (`len == 0`).

### Oneof
- Если в `oneof` не выбрано поле и **ровно одно** поле имеет `default_value` — оно будет выбрано и инициализировано.
- Если несколько полей в `oneof` имеют `default_value` — `Apply` вернёт ошибку.

### Рекурсия и циклы
- Библиотека рекурсивно обходит вложенные сообщения.
- Пустые вложенные сообщения инициализируются, только если в них (или их детях) есть хотя бы одно поле с дефолтом.
- Защита от бесконечной рекурсии в схеме реализована через отслеживание `FullName` дескрипторов.

## Генерация кода

Для использования кастомной опции вам понадобятся сгенерированные файлы. Рекомендуется использовать `buf` или `protoc`:

```bash
protoc -I proto --go_out=. --go_opt=module=github.com/yurasolovjov/protodefault proto/defaults/v1/options.proto
```
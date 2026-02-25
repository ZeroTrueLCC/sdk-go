# ZeroTrue Public API — Documentation for SDK Development

> **Version:** 1.0.0  
> **Base URL:** `https://<host>/api/v1`  
> **Protocol:** HTTPS + WebSocket (WSS)

---

## Table of Contents

1. [Overview & Architecture](#1-overview--architecture)
2. [Authentication](#2-authentication)
3. [Rate Limits & Quotas](#3-rate-limits--quotas)
4. [Endpoints](#4-endpoints)
   - [POST /analyze/file](#41-post-analyzefile)
   - [POST /analyze/text](#42-post-analyzetext)
   - [POST /analyze/url](#43-post-analyzeurl)
   - [GET /result/{content_id}](#44-get-resultcontent_id)
   - [GET /info](#45-get-info)
5. [Internal Backend Endpoints (proxied)](#5-internal-backend-endpoints-proxied)
   - [POST /check](#51-post-check)
   - [GET /checks/{check_id}/](#52-get-checkscheck_id)
6. [WebSocket: Real-time Results](#6-websocket-real-time-results)
7. [Data Models](#7-data-models)
8. [Error Handling](#8-error-handling)
9. [Idempotency](#9-idempotency)
10. [Supported Formats](#10-supported-formats)
11. [SDK Implementation Guide](#11-sdk-implementation-guide)

---

## 1. Overview & Architecture

ZeroTrue API — сервис анализа контента на AI-генерированность. Определяет вероятность того, что контент (текст, изображение, видео, аудио, код) создан ИИ.

### Архитектура (двухуровневая)

```
SDK Client
    │
    ▼
┌─────────────────────────────┐
│  API Gateway (FastAPI)      │  ← api_handler (порт 8001)
│  /api/v1/analyze/*          │     Синхронный API для SDK
│  /api/v1/result/*           │     Ожидает результат через WebSocket
│  /api/v1/info               │
└──────────┬──────────────────┘
           │ HTTP + WebSocket
           ▼
┌─────────────────────────────┐
│  Backend (Django REST)      │  ← основной бэкенд
│  /api/v1/check              │     Создание проверки → RabbitMQ → ML Worker
│  /api/v1/checks/{id}/       │     Получение результата
│  ws://host/ws/classification│     WebSocket уведомления
└─────────────────────────────┘
```

**Важно:** API Gateway (FastAPI) — это **синхронная обёртка**, которая:
1. Принимает запрос от клиента (file/text/url)
2. Проксирует его в Django Backend (`POST /api/v1/check`)
3. Подключается к WebSocket и **ожидает результат** (таймаут 5 минут)
4. Возвращает клиенту готовый результат

Альтернативно, SDK может работать **напрямую с Backend** (async flow):
1. `POST /api/v1/check` → получить `id` и статус `queued`
2. Поллинг `GET /api/v1/checks/{id}/` или подключение к WebSocket

---

## 2. Authentication

### API Key

Все запросы аутентифицируются через API ключ в заголовке `Authorization`.

```
Authorization: Bearer zt_<32_hex_characters>
```

#### Формат ключа

| Свойство | Значение |
|----------|----------|
| Префикс | `zt_` |
| Тело | UUID v4 без дефисов (32 hex символа) |
| Пример | `zt_a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6` |
| Хранение | SHA-256 хэш в БД (сырой ключ не хранится) |

#### Валидация на сервере

1. Извлекается значение после `Bearer `
2. Проверяется префикс `zt_`
3. Вычисляется `SHA-256(raw_key)` и ищется в таблице `api_keys`
4. Проверяется `is_active = true`
5. Обновляется `last_used_at`
6. Возвращается связанный `user`

#### Ошибки аутентификации

| HTTP Code | Описание |
|-----------|----------|
| `401` | Ключ отсутствует, неверный формат, или ключ не найден/деактивирован |
| `403` | Ключ валиден, но у пользователя нет прав |

---

## 3. Rate Limits & Quotas

### Rate Limits (per API key)

| Лимит | Значение | Период |
|-------|----------|--------|
| RPS | 60 запросов | в минуту |
| Daily | 10,000 запросов | в день |

При превышении RPS возвращается `429` (через DRF permission) с сообщением:
```
Rate limit exceeded: 60 requests per minute
```

При превышении daily лимита:
```
Daily limit exceeded: 10000 requests per day
```

### Credits (Quota)

Пользователь должен иметь кредиты для выполнения проверки.

| Тип скана | Списание из |
|-----------|-------------|
| Public scan (`is_private_scan=false`, `is_deep_scan=false`) | `user.credits` (бесплатные) |
| Private scan (`is_private_scan=true`) | `user.paid_credits` (платные) |
| Deep scan (`is_deep_scan=true`) | `user.paid_credits` (платные) |

Стоимость зависит от типа контента и определяется при создании `ContentItem`.

| HTTP Code | Error Code | Описание |
|-----------|-----------|----------|
| `400` | `INSUFFICIENT_CREDITS` | Нет бесплатных кредитов |
| `400` | `INSUFFICIENT_PAID_CREDITS` | Нет платных кредитов для private/deep scan |

---

## 4. Endpoints

### 4.1. POST /analyze/file

Загрузка и анализ файла. **Синхронный** — ожидает готовый результат (до 5 минут).

#### Request

```
POST /api/v1/analyze/file
Content-Type: multipart/form-data
```

| Поле | Тип | Обязательно | Default | Описание |
|------|-----|-------------|---------|----------|
| `file` | File | ✅ | — | Файл для анализа |
| `api_key` | string | ✅ | — | API ключ |
| `is_deep_scan` | boolean | ❌ | `false` | Глубокий анализ (платный) |
| `is_private_scan` | boolean | ❌ | `true` | Приватный скан (платный) |

#### Response `200 OK`

```json
{
  "id": "uuid-classification-result-id",
  "status": "completed",
  "result": { /* AnalysisResult */ }
}
```

---

### 4.2. POST /analyze/text

Анализ текста. **Синхронный**.

#### Request

```
POST /api/v1/analyze/text
Content-Type: multipart/form-data
```

| Поле | Тип | Обязательно | Default | Описание |
|------|-----|-------------|---------|----------|
| `text` | string | ✅ | — | Текст для анализа |
| `api_key` | string | ✅ | — | API ключ |
| `is_deep_scan` | boolean | ❌ | `false` | Глубокий анализ |
| `is_private_scan` | boolean | ❌ | `true` | Приватный скан |

#### Response `200 OK`

```json
{
  "id": "uuid-classification-result-id",
  "status": "completed",
  "result": { /* AnalysisResult */ }
}
```

---

### 4.3. POST /analyze/url

Анализ контента по URL. **Синхронный**.

#### Request

```
POST /api/v1/analyze/url
Content-Type: multipart/form-data
```

| Поле | Тип | Обязательно | Default | Описание |
|------|-----|-------------|---------|----------|
| `url` | string | ✅ | — | URL файла для анализа (http/https) |
| `api_key` | string | ✅ | — | API ключ |
| `is_deep_scan` | boolean | ❌ | `false` | Глубокий анализ |
| `is_private_scan` | boolean | ❌ | `true` | Приватный скан |

#### Валидация URL

- Только `http://` и `https://`
- Запрещены локальные адреса: `localhost`, `127.0.0.1`, `0.0.0.0`, `192.168.*`, `10.*`, `172.16-31.*`
- URL должен указывать на файл с допустимым расширением

#### Response `200 OK`

```json
{
  "id": "uuid-classification-result-id",
  "status": "completed",
  "result": { /* AnalysisResult */ }
}
```

---

### 4.4. GET /result/{content_id}

Получение результата по ID. Используется для повторного получения уже вычисленного результата.

#### Request

```
GET /api/v1/result/{content_id}?api_key=zt_xxx
```

| Параметр | Тип | Расположение | Обязательно | Описание |
|----------|-----|-------------|-------------|----------|
| `content_id` | string (UUID) | path | ✅ | ID из ответа `/analyze/*` |
| `api_key` | string | query | ✅ | API ключ |

#### Response `200 OK`

```json
{
  "id": "uuid",
  "status": "completed",
  "data": { /* AnalysisResult */ }
}
```

> **Примечание:** В `/analyze/*` результат в поле `result`, в `/result/{id}` — в поле `data`.

---

### 4.5. GET /info

Информация об API. **Не требует аутентификации.**

#### Response `200 OK`

```json
{
  "name": "ZeroTrue API",
  "version": "1.0.0",
  "description": "API for content analysis",
  "endpoints": {
    "analyze_file": "POST /analyze/file - Analyze uploaded file",
    "analyze_text": "POST /analyze/text - Analyze text",
    "analyze_url": "POST /analyze/url - Analyze content by URL",
    "get_result": "GET /result/{content_id} - Get result by ID",
    "api_info": "GET /info - API information"
  },
  "supported_formats": {
    "images": ["jpg", "jpeg", "png", "gif", "bmp", "tiff", "webp"],
    "videos": ["mp4", "mov", "avi", "mkv", "webm"],
    "audio": ["mp3", "wav", "ogg", "flac"],
    "code": ["py", "js", "html", "css", "java", "cpp", "go", "ts", "json"],
    "text": ["txt"]
  }
}
```

---

## 5. Internal Backend Endpoints (proxied)

Эти эндпоинты — внутренний Django Backend API. API Gateway проксирует к ним запросы. SDK может работать с ними напрямую для асинхронного flow.

### 5.1. POST /check

Создание проверки контента. Возвращает `202 Accepted` сразу (асинхронно).

#### Request — JSON (text/url)

```
POST /api/v1/check
Authorization: Bearer zt_xxx
Content-Type: application/json
```

```json
{
  "input": {
    "type": "text",       // "text" | "url" | "file"
    "value": "текст для проверки"
  },
  "is_deep_scan": false,
  "is_private_scan": true,
  "idempotency_key": "unique-key-123",  // optional, max 50 chars
  "metadata": {}                         // optional
}
```

#### Request — Multipart (file)

```
POST /api/v1/check
Authorization: Bearer zt_xxx
Content-Type: multipart/form-data
```

| Поле | Тип | Описание |
|------|-----|----------|
| `input_file` | File | Файл для анализа |
| `input_type` | string | `"file"` |
| `is_deep_scan` | boolean | Глубокий анализ |
| `is_private_scan` | boolean | Приватный скан |

**Альтернативно:** вложенная структура `input` или плоские параметры `input_type` + `input_value`/`input_file` (нельзя использовать одновременно).

#### Response `202 Accepted`

```json
{
  "id": "uuid-content-item-id",
  "status": "queued"
}
```

> **Важно:** `id` в ответе — это `ContentItem.id`. В WebSocket и `/checks/{id}` результат будет привязан к этому ID.

---

### 5.2. GET /checks/{check_id}/

Получение статуса и результата проверки.

#### Request

```
GET /api/v1/checks/{check_id}/
Authorization: Bearer zt_xxx
```

| Параметр | Тип | Описание |
|----------|-----|----------|
| `check_id` | UUID | ID из ответа `POST /check` или `ClassificationResult.id` |

- Сервер ищет сначала по `ContentItem.id`, затем по `ClassificationResult.id`
- Проверяется принадлежность контента текущему пользователю

#### Response `200 OK` (completed)

```json
{
  "id": "check_id",
  "status": "completed",
  "result": {
    "ai_probability": 0.85,
    "human_probability": 0.15,
    "combined_probability": 0.85,
    "result_type": "text_analysis",
    "ml_model": "model_name",
    "ml_model_version": "1.0.0",
    "details": { /* raw ML details */ },
    "feedback": null,
    "created_at": "2025-01-15T10:30:00Z",
    "status": "completed",
    "file_url": "https://cdn.example.com/files/abc.png",
    "original_filename": "document.pdf",
    "size_bytes": 1048576,
    "size_mb": 1.0,
    "resolution": "1920x1080",
    "length": 500,
    "content": "текст контента (для text типа)",
    "is_private_scan": true,
    "is_deep_scan": false,
    "price": 1,
    "preview_url": "https://cdn.example.com/previews/abc.jpg",
    "inference_time_ms": 1234,
    "api_schema_version": "1.0.0",
    "meta_mime": "image/png",
    "meta_file_size_bytes": 1048576,
    "meta_sha256": "e3b0c44298fc1c149afbf4c8996fb924...",
    "meta_content_url": "https://...",
    "meta_content_type": "image",
    "details_summary": {
      "overall_assessment": "AI-generated",
      "processing_time_s": 1.234,
      "gen_technique": "diffusion"
    },
    "details_extra": { /* модально-специфичные данные */ },
    "suspected_models": [
      {
        "model_name": "ChatGPT",
        "confidence_pct": 95.5
      },
      {
        "model_name": "Claude",
        "confidence_pct": 3.2
      }
    ],
    "segments": [
      {
        "label": "aigen",
        "confidence_pct": 92.0,
        "start_char": 0,
        "end_char": 150,
        "start_line": null,
        "end_line": null,
        "start_s": null,
        "end_s": null,
        "timecode": null
      }
    ],
    "views_count": 42
  }
}
```

#### Response (pending/processing)

```json
{
  "id": "check_id",
  "status": "processing",
  "result": {
    "created_at": "2025-01-15T10:30:00Z",
    "status": "processing"
  }
}
```

---

## 6. WebSocket: Real-time Results

WebSocket используется для получения результата в реальном времени.

### Подключение

```
wss://<host>/ws/classification/{content_item_id}/
```

| Header | Описание |
|--------|----------|
| `Authorization` | `Bearer zt_xxx` (опционально, для аутентификации) |

### Поведение

1. Клиент подключается к WebSocket по `content_item_id` (из ответа `POST /check`)
2. Если результат уже готов — сервер отправляет его **сразу при подключении**
3. Иначе — сервер отправляет результат когда ML worker завершит обработку
4. Клиент получает **одно сообщение** с полным результатом

### Формат сообщения (JSON)

```json
{
  "id": "uuid-classification-result-id",
  "ai_probability": 0.85,
  "human_probability": 0.15,
  "combined_probability": 0.85,
  "result_type": "text_analysis",
  "ml_model": "model_name",
  "content_item_status": "completed",
  "content_item_url": "https://...",
  "content_item_original_filename": "doc.pdf",
  "content_item_size_bytes": 1048576,
  "content_item_size_mb": 1.0,
  "content_item_is_private_scan": true,
  "content_item_is_deep_scan": false,
  "content_item_price": 1,
  "inference_time_ms": 1234,
  "suspected_models": [...],
  "segments": [...]
}
```

> **Примечание:** В WebSocket поля ContentItem имеют префикс `content_item_*`. API Gateway переименовывает их в короткие имена перед отдачей клиенту.

### Ping/Pong

Для поддержания соединения клиент может отправить:
```json
{"type": "ping"}
```
Сервер ответит:
```json
{"type": "pong"}
```

### Таймаут

API Gateway ожидает результат по WebSocket **до 5 минут** (300 секунд). При превышении возвращается `408 Request Timeout`.

---

## 7. Data Models

### AnalysisResult

Полный результат анализа, возвращаемый всеми эндпоинтами.

| Поле | Тип | Nullable | Описание |
|------|-----|----------|----------|
| `ai_probability` | float | ❌ | Вероятность AI (0.0–1.0) |
| `human_probability` | float | ❌ | Вероятность Human (0.0–1.0) |
| `combined_probability` | float | ❌ | Комбинированная вероятность (0.0–1.0) |
| `result_type` | string | ❌ | Тип результата анализа |
| `ml_model` | string | ❌ | Название ML модели |
| `ml_model_version` | string | ✅ | Версия ML модели |
| `details` | object | ✅ | Детали анализа (модально-специфичные) |
| `feedback` | string | ✅ | Обратная связь пользователя |
| `created_at` | string (ISO 8601) | ✅ | Время создания результата |
| `status` | string | ✅ | Статус контент-элемента: `pending`, `processing`, `completed`, `failed` |
| `file_url` | string (URL) | ✅ | URL загруженного файла |
| `original_filename` | string | ✅ | Оригинальное имя файла |
| `size_bytes` | integer | ✅ | Размер файла в байтах |
| `size_mb` | float | ✅ | Размер файла в МБ |
| `resolution` | string | ✅ | Разрешение изображения/видео (например `"1920x1080"`) |
| `length` | integer | ✅ | Длина контента (символы для текста, секунды для медиа) |
| `content` | string | ✅ | Текстовый контент (для type=text) |
| `is_private_scan` | boolean | ✅ | Приватный скан |
| `is_deep_scan` | boolean | ✅ | Глубокий скан |
| `price` | integer | ✅ | Стоимость проверки в кредитах |
| `inference_time_ms` | integer | ✅ | Время выполнения ML модели в мс |
| `api_schema_version` | string | ✅ | Версия схемы API (например `"1.0.0"`) |
| `meta_mime` | string | ✅ | MIME-тип файла |
| `meta_file_size_bytes` | integer | ✅ | Размер файла из метаданных |
| `meta_sha256` | string | ✅ | SHA-256 хэш файла |
| `meta_content_url` | string | ✅ | URL контента из метаданных |
| `meta_content_type` | string | ✅ | Тип контента из метаданных |
| `details_summary` | object | ✅ | Суммарная оценка (см. ниже) |
| `details_extra` | object | ✅ | Дополнительные детали по типу контента |
| `suspected_models` | array | ✅ | Предполагаемые AI-модели генераторы |
| `segments` | array | ✅ | Сегменты анализа |

### ContentType (enum)

| Значение | Описание |
|----------|----------|
| `text` | Текст |
| `image` | Изображение |
| `video` | Видео |
| `code` | Код |
| `voice` | Голос |
| `music` | Музыка |

### Status (enum)

| Значение | Описание |
|----------|----------|
| `pending` | В очереди |
| `processing` | Обрабатывается |
| `completed` | Завершено |
| `failed` | Ошибка |

### SuspectedModel

| Поле | Тип | Описание |
|------|-----|----------|
| `model_name` | string | Название модели (ChatGPT, Claude, Gemini, Midjourney, etc.) |
| `confidence_pct` | float | Уверенность в процентах (0–100) |

### Segment

Сегмент анализа — часть контента с отдельной оценкой.

| Поле | Тип | Описание |
|------|-----|----------|
| `label` | string | Метка: `aigen`, `human`, `deepfake`, `authentic`, etc. |
| `confidence_pct` | float | Уверенность (0–100) |
| `start_char` | integer? | Начало сегмента (символ, для text/code) |
| `end_char` | integer? | Конец сегмента (символ, для text/code) |
| `start_line` | integer? | Начальная строка (для code) |
| `end_line` | integer? | Конечная строка (для code) |
| `start_s` | float? | Начало в секундах (для video/voice/music) |
| `end_s` | float? | Конец в секундах (для video/voice/music) |
| `timecode` | string? | Таймкод (для video/voice/music) |

### DetailsSummary (типичная структура)

```json
{
  "overall_assessment": "AI-generated",
  "processing_time_s": 1.234,
  "gen_technique": "diffusion"
}
```

---

## 8. Error Handling

### Формат ошибки

Все ошибки возвращаются в стандартном формате:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable description"
  },
  "request_id": "req_a1b2c3d4e5f6a7b8"
}
```

### Коды ошибок

| HTTP Code | Error Code | Описание |
|-----------|-----------|----------|
| `400` | `INVALID_FILE` | Невалидный файл (повреждён, не может быть обработан) |
| `400` | `INSUFFICIENT_CREDITS` | Нет бесплатных кредитов |
| `400` | `INSUFFICIENT_PAID_CREDITS` | Нет платных кредитов |
| `400` | `BAD_REQUEST` | Невалидный UUID формат |
| `401` | `AUTHENTICATION_FAILED` | Невалидный API ключ |
| `403` | `FORBIDDEN` | Доступ к файлу запрещён |
| `404` | `NOT_FOUND` | Файл / проверка не найдена |
| `408` | `TIMEOUT` | Таймаут при скачивании файла по URL |
| `422` | `VALIDATION_ERROR` | Ошибка валидации входных данных |
| `429` | — | Превышен rate limit (RPS или daily) |
| `500` | `INTERNAL` | Внутренняя ошибка сервера |
| `502` | `BAD_GATEWAY` | Ошибка подключения к внешнему ресурсу |

### Валидационные ошибки (422)

- `Multiple files upload is not supported. Please upload one file at a time`
- `Either 'input' object or flat parameters (input_type, input_value, input_file) must be provided`
- `Cannot use both nested 'input' and flat parameters`
- `Text value is required for text type`
- `URL value is required for url type`
- `Invalid URL format`
- `Only HTTP/HTTPS URLs are allowed`
- `Local addresses are not allowed`
- `File is required for file type`
- `Unsupported file format`
- `Invalid image file`
- `URL must point to a file with valid extension`

---

## 9. Idempotency

API поддерживает идемпотентность для предотвращения дублирования запросов.

### Использование

Передайте `idempotency_key` в теле запроса `POST /check`:

```json
{
  "input": {"type": "text", "value": "..."},
  "idempotency_key": "unique-operation-id-123"
}
```

### Поведение

| Ситуация | Действие |
|----------|---------|
| Ключ новый | Запрос обрабатывается, результат кэшируется |
| Ключ существует + не истёк | Возвращается **сохранённый ответ** без повторной обработки |
| Ключ истёк (>24ч) | Запрос обрабатывается как новый |

### Ограничения

- Максимальная длина ключа: **50 символов**
- TTL записи: **24 часа**
- Уникальность в рамках одного API ключа

---

## 10. Supported Formats

### По типам контента

| Тип | Расширения |
|-----|-----------|
| **Images** | jpg, jpeg, png, gif, bmp, tiff, webp |
| **Videos** | mp4, mov, avi, mkv, webm |
| **Audio** | mp3, wav, ogg, flac |
| **Code** | py, js, html, css, java, cpp, go, ts, json |
| **Text** | txt |

---

## 11. SDK Implementation Guide

### Рекомендуемые стратегии

#### Стратегия 1: Синхронный API (простой, через Gateway)

SDK вызывает `POST /api/v1/analyze/{file|text|url}` и получает готовый результат. Подходит для простых интеграций.

```
Client → POST /analyze/file → [ожидание до 5 мин] → Response с результатом
```

**Плюсы:** Простота, один запрос.  
**Минусы:** Долгие таймауты, блокирующий запрос.

#### Стратегия 2: Асинхронный API (через Backend напрямую)

SDK работает с backend напрямую: создаёт проверку и поллит результат.

```
Client → POST /check → 202 {id, status: "queued"}
         ↓
Client → GET /checks/{id}/ → 200 {status: "processing"} (поллинг)
         ↓
Client → GET /checks/{id}/ → 200 {status: "completed", result: {...}}
```

**Рекомендуемый интервал поллинга:** 1.5 секунды (из `poll_after_seconds`).

#### Стратегия 3: WebSocket (наиболее эффективная)

SDK создаёт проверку и подключается к WebSocket для мгновенного получения результата.

```
Client → POST /check → 202 {id, status: "queued"}
         ↓
Client → WSS /ws/classification/{id}/ → получает результат мгновенно
```

### Структура SDK

```
sdk/
├── client.go (или client.py, client.ts)     # Основной клиент
│   ├── NewClient(apiKey, options)            # Конструктор
│   ├── AnalyzeFile(file, options)            # Анализ файла
│   ├── AnalyzeText(text, options)            # Анализ текста
│   ├── AnalyzeURL(url, options)              # Анализ по URL
│   ├── GetResult(id)                         # Получение результата
│   └── GetInfo()                             # Информация об API
│
├── models.go                                 # Типы данных
│   ├── AnalysisResult                        # Результат анализа
│   ├── SuspectedModel                        # Предполагаемая AI модель
│   ├── Segment                               # Сегмент анализа
│   ├── ContentResponse                       # Обёртка ответа
│   ├── ErrorResponse                         # Ошибка
│   └── APIInfo                               # Информация об API
│
├── options.go                                # Опции запросов
│   ├── AnalyzeOptions                        # is_deep_scan, is_private_scan
│   └── ClientOptions                         # baseURL, timeout, retries
│
└── errors.go                                 # Типизированные ошибки
    ├── AuthenticationError                   # 401
    ├── RateLimitError                        # 429
    ├── InsufficientCreditsError              # 400 + INSUFFICIENT_CREDITS
    ├── ValidationError                       # 422
    ├── NotFoundError                         # 404
    └── TimeoutError                          # 408
```

### Рекомендации для SDK

1. **Retry logic:** Автоматический повтор при 5xx ошибках с exponential backoff
2. **Rate limit handling:** При 429 — ожидание и повтор
3. **Timeout:** Настраиваемый таймаут (default 5 минут для analyze, 30 секунд для get)
4. **Idempotency:** Автоматическая генерация `idempotency_key` при retry
5. **Content-Type detection:** Автоматическое определение типа файла при загрузке
6. **Streaming:** Поддержка больших файлов через `multipart/form-data`
7. **WebSocket reconnect:** Автоматическое переподключение при обрыве

### Пример использования (Go)

```go
client := zerotrue.NewClient("zt_your_api_key")

// Синхронный анализ файла
result, err := client.AnalyzeFile("image.png", &zerotrue.AnalyzeOptions{
    IsDeepScan:    false,
    IsPrivateScan: true,
})
if err != nil {
    log.Fatal(err)
}

fmt.Printf("AI Probability: %.2f%%\n", result.AIprobability * 100)
fmt.Printf("Model: %s\n", result.MLModel)

for _, model := range result.SuspectedModels {
    fmt.Printf("  %s: %.1f%%\n", model.ModelName, model.ConfidencePct)
}

// Асинхронный flow
check, _ := client.CreateCheck("Some text to analyze", nil)
// check.ID = "uuid", check.Status = "queued"

result, _ = client.GetResult(check.ID)
// result.Status = "completed"
```

---

## Appendix A: Field Mapping (WebSocket → API)

API Gateway переименовывает следующие поля из WebSocket ответа:

| WebSocket (оригинал) | API (переименованное) |
|----------------------|----------------------|
| `content_item_status` | `status` |
| `content_item_url` | `file_url` |
| `content_item_original_filename` | `original_filename` |
| `content_item_size_bytes` | `size_bytes` |
| `content_item_size_mb` | `size_mb` |
| `content_item_resolution` | `resolution` |
| `content_item_length` | `length` |
| `content_item_content` | `content` |
| `content_item_is_private_scan` | `is_private_scan` |
| `content_item_is_deep_scan` | `is_deep_scan` |
| `content_item_price` | `price` |

## Appendix B: Два формата input в POST /check

Сериализатор поддерживает два взаимоисключающих формата:

**Вариант 1: Вложенная структура (рекомендуемая)**
```json
{
  "input": {
    "type": "text",
    "value": "текст"
  }
}
```

**Вариант 2: Плоские параметры**
```
input_type=text
input_value=текст
```
или для файлов:
```
input_type=file
input_file=@file.png
```

Оба формата нельзя использовать одновременно — сервер вернёт `422`.

# Annotation Service - LLM-powered Dataset Creator

Микросервис на Go для автоматической разметки датасета с помощью Gemini API.

## Возможности

- 🤖 **Gemini API Integration**: Использует бесплатный Gemini 2.0 Flash (1500 req/day)
- 📊 **9 категорий угроз**: Классификация по всем категориям из вашего llm.py
- 🔄 **Batch Processing**: Асинхронная обработка больших датасетов
- 💾 **SQLite Storage**: Хранение размеченных данных
- 📤 **Export**: CSV/JSON для fine-tuning DistilBERT
- ⚡ **Fast**: Go + concurrency для высокой производительности

## API Endpoints

### Annotation

```bash
# Single message
POST /api/v1/annotate/single
{
  "text": "СРОЧНО! Кликни здесь для приза!"
}

# Batch messages
POST /api/v1/annotate/batch
{
  "messages": [
    {"id": 1, "text": "message 1"},
    {"id": 2, "text": "message 2"}
  ]
}

# Check batch job status
GET /api/v1/annotate/jobs/:job_id
```

### Data Retrieval

```bash
# Get all annotations
GET /api/v1/annotations

# Get by category (1-9)
GET /api/v1/annotations/category/:id

# Get statistics
GET /api/v1/annotations/stats
```

### Export

```bash
# Export to CSV (for fine-tuning)
GET /api/v1/export/csv

# Export to JSON
GET /api/v1/export/json
```

## Быстрый старт

### 1. Настройка

Скопируйте ваш API ключ из llm.py в config:

```bash
cd social-engineering-detector/annotation-service

# Отредактировать configs/config.yml
```

```yaml
gemini:
  api_key: "YOUR_GEMINI_API_KEY"  # Из llm.py: AIzaSyCUphSo3aAhaw7ndxpz8hOBsco52UQMkPs
```

### 2. Установка зависимостей

```bash
go mod download
```

### 3. Запуск

```bash
go run cmd/server/main.go
```

Сервис запустится на `http://localhost:8002`

## Использование

### Разметка одного сообщения

```bash
curl -X POST http://localhost:8002/api/v1/annotate/single \
  -H "Content-Type: application/json" \
  -d '{"text": "СРОЧНО! Ваш аккаунт будет заблокирован!"}'
```

**Response:**
```json
{
  "id": 1,
  "text": "СРОЧНО! Ваш аккаунт будет заблокирован!",
  "category_id": 7,
  "category_name": "Финансовое мошенничество",
  "justification": "Сообщение содержит urgency и угрозу блокировки...",
  "confidence": 0.95,
  "annotated_at": "2025-10-23T...",
  "provider": "gemini",
  "model_version": "gemini-2.0-flash-exp"
}
```

### Batch разметка

```bash
curl -X POST http://localhost:8002/api/v1/annotate/batch \
  -H "Content-Type: application/json" \
  -d @messages.json
```

**messages.json:**
```json
{
  "messages": [
    {"id": 1, "text": "Привет, как дела?"},
    {"id": 2, "text": "Кликни на ссылку для приза!"},
    {"id": 3, "text": "Отправь мне пароль"}
  ]
}
```

**Response:**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "message": "Batch annotation started..."
}
```

**Проверка статуса:**
```bash
curl http://localhost:8002/api/v1/annotate/jobs/550e8400-e29b-41d4-a716-446655440000
```

### Экспорт для fine-tuning

```bash
# Скачать CSV
curl http://localhost:8002/api/v1/export/csv -o dataset.csv

# Формат CSV:
# text,category_id,category_name,justification
```

## Workflow создания датасета

### 1. Подготовка сообщений

Создайте файл с сообщениями для разметки:

```json
{
  "messages": [
    {"text": "message 1"},
    {"text": "message 2"},
    ...
    {"text": "message 2000"}
  ]
}
```

### 2. Запуск batch разметки

```bash
curl -X POST http://localhost:8002/api/v1/annotate/batch \
  -H "Content-Type: application/json" \
  -d @messages_to_annotate.json
```

### 3. Мониторинг прогресса

```bash
# Проверка статистики
curl http://localhost:8002/api/v1/annotations/stats

# Пример ответа:
{
  "total": 1523,
  "by_category": {
    "Нейтральное общение": 1200,
    "Фишинг": 150,
    "Финансовое мошенничество": 100,
    ...
  }
}
```

### 4. Экспорт и fine-tuning

```bash
# Экспорт в CSV
curl http://localhost:8002/api/v1/export/csv -o training_dataset.csv

# Использовать в ML-service для fine-tuning
cd ../ml-service
python train.py --dataset ../annotation-service/training_dataset.csv
```

## Rate Limiting

Gemini Free Tier: **1500 запросов/день**

Сервис автоматически добавляет задержку 100ms между запросами в batch режиме.

**Максимум в день:**
- Single: 1500 сообщений
- Batch: зависит от размера (рекомендуется < 1000/день для запаса)

## Структура данных

### 9 категорий угроз

| ID | Название |
|----|----------|
| 1  | Склонение к сексуальным действиям (Груминг) |
| 2  | Угрозы, шантаж, вымогательство |
| 3  | Физическое насилие/Буллинг |
| 4  | Склонение к суициду/Самоповреждению |
| 5  | Склонение к опасным играм/действиям |
| 6  | Пропаганда запрещенных веществ |
| 7  | Финансовое мошенничество |
| 8  | Сбор личных данных (Фишинг) |
| 9  | Нейтральное общение |

### Database Schema

**annotations table:**
- id: INTEGER PRIMARY KEY
- text: TEXT (исходное сообщение)
- category_id: INTEGER (1-9)
- category_name: TEXT (название категории)
- justification: TEXT (объяснение от Gemini)
- confidence: REAL (уверенность модели)
- annotated_at: DATETIME
- provider: TEXT ("gemini")
- model_version: TEXT
- is_validated: BOOLEAN (для ручной проверки)

## Производительность

- **Latency**: ~1-2 секунды на сообщение (Gemini API)
- **Throughput**: ~50-100 сообщений/минута (с rate limiting)
- **Memory**: ~50 MB
- **Storage**: ~1 KB на аннотацию (SQLite)

## Troubleshooting

### API Key Error

```
Gemini API key not configured
```

**Решение**: Установите `api_key` в `configs/config.yml`

### Rate Limit Exceeded

```
429 Too Many Requests
```

**Решение**: Подождите до следующего дня или увеличьте delay между запросами

### Inconsistent Categories

Если Gemini возвращает неправильные категории:

1. Проверьте промты в `internal/gemini/prompts.go`
2. Добавьте больше few-shot примеров
3. Используйте более мощную модель (gemini-1.5-pro)

## Интеграция с ML Service

После создания датасета:

```bash
# 1. Экспорт
curl http://localhost:8002/api/v1/export/csv -o dataset.csv

# 2. Fine-tuning (в ml-service)
cd ../ml-service
python -c "
from app.models.model_loader import ClassBalancedTrainer
import pandas as pd

# Загрузить датасет
df = pd.read_csv('../annotation-service/dataset.csv')
texts = df['text'].tolist()
labels = ['attack' if cat != 9 else 'non-attack' for cat in df['category_id']]

# Обучить модель
trainer = ClassBalancedTrainer()
model = trainer.train(
    train_texts=texts,
    train_labels=labels,
    output_dir='./models/distilbert-se-detector',
    num_epochs=3
)
"

# 3. Перезапустить ML Service
python -m app.main
```

## Roadmap

- [ ] Multi-provider support (Groq, OpenRouter)
- [ ] Manual validation UI
- [ ] Active learning (отбор трудных примеров)
- [ ] Confidence calibration
- [ ] Export to Hugging Face datasets format

## License

MIT

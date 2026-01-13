# ML Service - Social Engineering Detection

Микросервис для детекции социальной инженерии в текстовых сообщениях на основе fine-tuned DistilBERT модели.

## Возможности

- **Бинарная классификация**: `attack` / `non-attack`
- **Поддержка русского языка**: multilingual DistilBERT
- **Батчинг**: эффективная обработка множества сообщений
- **Балансировка классов**: weighted loss для работы с несбалансированными данными
- **Self-hosted**: полностью автономная работа без внешних API
- **Fast API**: REST API с автоматической документацией

## Архитектура

```
┌─────────────────────────────────────────┐
│         FastAPI Application             │
├─────────────────────────────────────────┤
│  Preprocessing (TextPreprocessor)       │
│   - Normalization                       │
│   - Tokenization (WordPiece)            │
│   - Padding/Truncation                  │
├─────────────────────────────────────────┤
│  Model (SEClassifier)                   │
│   - DistilBERT (6 layers)               │
│   - Binary Classification Head          │
│   - Weighted Loss (class balancing)     │
├─────────────────────────────────────────┤
│  API Endpoints                          │
│   - POST /api/v1/classify/single        │
│   - POST /api/v1/classify/batch         │
│   - GET  /api/v1/health                 │
│   - GET  /api/v1/model/info             │
└─────────────────────────────────────────┘
```

## Быстрый старт

### 1. Установка зависимостей

```bash
cd social-engineering-detector/ml-service
pip install -r requirements.txt
```

### 2. Конфигурация

Скопируйте `.env.example` в `.env` и настройте параметры:

```bash
cp .env.example .env
```

Основные параметры:
- `DEVICE`: `cpu` или `cuda` (если есть GPU)
- `MODEL_NAME`: `distilbert-base-multilingual-cased`
- `CONFIDENCE_THRESHOLD`: порог уверенности (default: 0.5)

### 3. Запуск сервиса

#### Локально (для разработки)

```bash
python -m app.main
```

Сервис будет доступен на `http://localhost:8001`

#### Docker

```bash
# Создать сеть (если еще не создана)
docker network create se-detector-network

# Запустить сервис
docker-compose up -d
```

#### Проверка работоспособности

```bash
curl http://localhost:8001/api/v1/health
```

## API Endpoints

### 1. Классификация одного сообщения

```bash
POST /api/v1/classify/single
Content-Type: application/json

{
  "text": "СРОЧНО! Ваш аккаунт будет заблокирован. Нажмите сюда для подтверждения."
}
```

**Ответ:**
```json
{
  "category": "attack",
  "confidence": 0.9523,
  "probabilities": {
    "non-attack": 0.0477,
    "attack": 0.9523
  },
  "is_attack": true,
  "threshold": 0.5,
  "processing_time_ms": 45.32
}
```

### 2. Батчинг (множество сообщений)

```bash
POST /api/v1/classify/batch
Content-Type: application/json

{
  "messages": [
    {"id": 1, "text": "Привет, как дела?"},
    {"id": 2, "text": "Кликни на ссылку и получи приз!"},
    {"id": 3, "text": "Завтра встречаемся?"}
  ]
}
```

**Ответ:**
```json
{
  "results": [
    {
      "id": 1,
      "category": "non-attack",
      "confidence": 0.9812,
      "is_attack": false,
      ...
    },
    {
      "id": 2,
      "category": "attack",
      "confidence": 0.8934,
      "is_attack": true,
      ...
    },
    {
      "id": 3,
      "category": "non-attack",
      "confidence": 0.9645,
      "is_attack": false,
      ...
    }
  ],
  "total": 3,
  "processing_time_ms": 112.45
}
```

### 3. Информация о модели

```bash
GET /api/v1/model/info
```

**Ответ:**
```json
{
  "service_name": "ml-service",
  "version": "1.0.0",
  "model_name": "distilbert-base-multilingual-cased",
  "num_labels": 2,
  "labels": ["non-attack", "attack"],
  "device": "cpu",
  "max_length": 512
}
```

## Обучение модели

### Подготовка датасета

Требуется CSV/JSON файл с колонками:
- `text`: текст сообщения
- `label`: метка (`attack` или `non-attack`)

Пример:
```csv
text,label
"Привет, как дела?",non-attack
"СРОЧНО! Кликни здесь!",attack
```

### Обучение

```python
from app.models.model_loader import ClassBalancedTrainer

# Ваши данные
train_texts = ["text1", "text2", ...]
train_labels = ["non-attack", "attack", ...]

# Инициализация
trainer = ClassBalancedTrainer()

# Обучение с балансировкой классов
model = trainer.train(
    train_texts=train_texts,
    train_labels=train_labels,
    eval_texts=eval_texts,  # опционально
    eval_labels=eval_labels,
    output_dir="./models/distilbert-se-detector",
    num_epochs=3,
    batch_size=16,
    learning_rate=2e-5
)
```

### Метрики

После обучения модель автоматически сохраняется в `models/distilbert-se-detector/`. При следующем запуске сервис будет использовать обученную модель.

Ожидаемые метрики (по статье 2025):
- **Accuracy**: >95%
- **F1-score**: >0.95
- **Precision**: >0.94
- **Recall**: >0.96

## Интеграция с Backend

### Пример на Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

type ClassifyRequest struct {
    Text string `json:"text"`
}

type ClassifyResponse struct {
    Category    string  `json:"category"`
    Confidence  float64 `json:"confidence"`
    IsAttack    bool    `json:"is_attack"`
}

func classifyMessage(text string) (*ClassifyResponse, error) {
    url := "http://ml-service:8001/api/v1/classify/single"

    reqBody, _ := json.Marshal(ClassifyRequest{Text: text})
    resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result ClassifyResponse
    json.NewDecoder(resp.Body).Decode(&result)

    return &result, nil
}
```

### Пример на Python

```python
import requests

def classify_message(text: str) -> dict:
    url = "http://localhost:8001/api/v1/classify/single"
    response = requests.post(url, json={"text": text})
    return response.json()

result = classify_message("СРОЧНО! Ваш аккаунт заблокирован!")
print(f"Attack: {result['is_attack']}, Confidence: {result['confidence']}")
```

## Производительность

### Латентность (CPU, single message)
- Preprocessing: ~5ms
- Inference: ~30-50ms
- Total: ~35-55ms

### Throughput (batch processing)
- Batch 16: ~200-300 msg/sec
- Batch 32: ~350-500 msg/sec

### GPU ускорение
Для использования GPU:
1. Установить `torch` с CUDA support
2. Изменить `DEVICE=cuda` в `.env`
3. Ожидаемое ускорение: 3-5x

## Мониторинг

### Health Check

```bash
curl http://localhost:8001/api/v1/health
```

### Logs

```bash
# Docker
docker logs -f se-detector-ml-service

# Local
# Логи выводятся в stdout
```

### Метрики (TODO)
- Prometheus endpoint для метрик
- Grafana dashboard

## Troubleshooting

### Модель не загружается
- Проверьте, что `models/distilbert-se-detector/` существует
- Если нет, сервис использует pre-trained модель (ниже точность)
- Обучите модель или скачайте готовую

### Out of Memory
- Уменьшите `BATCH_SIZE` в `.env`
- Уменьшите `MAX_LENGTH` (default: 512)
- Используйте CPU вместо GPU для малых нагрузок

### Низкая точность
- Убедитесь, что модель обучена на релевантном датасете
- Проверьте балансировку классов
- Увеличьте размер обучающего датасета

## Roadmap

- [ ] Multi-class классификация (9 категорий угроз)
- [ ] Поддержка других языков (английский, украинский)
- [ ] Экспорт в ONNX для production
- [ ] A/B testing framework
- [ ] Model versioning

## Лицензия

MIT

## Контакты

Вопросы и предложения: [создать issue](https://github.com/...)

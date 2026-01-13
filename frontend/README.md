# SE Detector Frontend

Веб-интерфейс для системы обнаружения социальной инженерии.

## Технологии

- **React 18** - UI библиотека
- **Vite** - Сборщик и dev server
- **Tailwind CSS** - Utility-first CSS фреймворк
- **React Router** - Роутинг
- **Axios** - HTTP клиент
- **Lucide React** - Иконки
- **Recharts** - Графики и диаграммы
- **date-fns** - Работа с датами

## Структура проекта

```
frontend/
├── src/
│   ├── components/      # Переиспользуемые компоненты
│   │   ├── Layout.jsx   # Основной layout с sidebar
│   │   └── PrivateRoute.jsx  # Защищенный роут
│   ├── contexts/        # React Context
│   │   └── AuthContext.jsx   # Контекст авторизации
│   ├── pages/          # Страницы приложения
│   │   ├── Login.jsx   # Страница входа
│   │   ├── Register.jsx # Страница регистрации
│   │   ├── Dashboard.jsx # Главный дашборд
│   │   ├── Incidents.jsx # Список инцидентов
│   │   ├── Chats.jsx   # Управление чатами
│   │   └── Analytics.jsx # Аналитика и графики
│   ├── services/       # API сервисы
│   │   └── api.js      # Axios конфигурация и API методы
│   ├── styles/         # Стили
│   │   └── index.css   # Tailwind и кастомные стили
│   ├── App.jsx         # Главный компонент
│   └── main.jsx        # Точка входа
├── Dockerfile          # Docker образ для production
├── nginx.conf          # Nginx конфигурация
├── package.json        # Зависимости
└── vite.config.js      # Vite конфигурация
```

## Запуск для разработки

### Локально (без Docker)

```bash
# Установить зависимости
npm install

# Запустить dev server
npm run dev

# Приложение будет доступно на http://localhost:3000
```

### С Docker

```bash
# Из корня проекта
docker-compose up frontend

# Приложение будет доступно на http://localhost:3000
```

## Production build

```bash
# Собрать production версию
npm run build

# Результат в папке dist/
```

## Особенности

### Авторизация

- JWT токены в localStorage
- Автоматический редирект на /login при 401
- Защищенные роуты через PrivateRoute компонент
- Контекст авторизации через React Context API

### API интеграция

Все API запросы проксируются через Nginx в Docker:
```
Frontend (http://localhost:3000)
    ↓
Nginx proxy /api
    ↓
Backend (http://backend:8080)
```

### Дизайн

Лаконичный минималистичный дизайн:
- Светлая цветовая схема (gray-50 фон)
- Четкая типография
- Карточный layout
- Современные UI компоненты
- Адаптивная вёрстка (mobile-friendly)

### Компоненты

#### Dashboard
- Ключевые метрики (карточки со статистикой)
- Последние инциденты
- Распределение по категориям угроз

#### Incidents
- Список всех инцидентов с пагинацией
- Фильтрация по категориям
- Поиск по тексту
- Цветовые метки для разных типов угроз

#### Chats
- Список мониторимых чатов
- Статистика по каждому чату
- Статус мониторинга

#### Analytics
- Графики и диаграммы (Recharts)
- Pie chart - распределение по категориям
- Bar chart - количество инцидентов
- Line chart - тренды
- Детальная таблица статистики

## Цветовая схема

```css
Primary (Blue):   #0ea5e9
Danger (Red):     #ef4444
Warning (Yellow): #f59e0b
Success (Green):  #22c55e
```

## Категории угроз

| ID | Название | Цвет |
|----|----------|------|
| 1  | Груминг | danger (красный) |
| 2  | Шантаж | danger (красный) |
| 3  | Буллинг | warning (желтый) |
| 4  | Склонение к суициду | danger (красный) |
| 5  | Опасные игры | danger (красный) |
| 6  | Пропаганда веществ | warning (желтый) |
| 7  | Финансовое мошенничество | warning (желтый) |
| 8  | Фишинг | warning (желтый) |

## Environment переменные

```env
VITE_API_URL=http://localhost:8080
```

## Nginx конфигурация

- Gzip сжатие
- Кеширование статических файлов (1 год)
- Proxy для /api на backend:8080
- SPA fallback (все роуты → index.html)

## TODO

- [ ] Websockets для real-time уведомлений
- [ ] Dark mode
- [ ] Экспорт данных (CSV, PDF)
- [ ] Настройки пользователя
- [ ] Уведомления (toast notifications)
- [ ] Детальная страница инцидента
- [ ] Bulk операции над инцидентами

## Скриншоты

### Login
![Login](docs/screenshots/login.png)

### Dashboard
![Dashboard](docs/screenshots/dashboard.png)

### Incidents
![Incidents](docs/screenshots/incidents.png)

### Analytics
![Analytics](docs/screenshots/analytics.png)

# OneWay Support Bot 2.0

[![Go Report Card](https://goreportcard.com/badge/github.com/memrook/oneway_subot?style=flat-square)](https://goreportcard.com/report/github.com/memrook/oneway_subot)
[![Docker](https://img.shields.io/docker/automated/memrook/oneway_subot.svg?style=flat-square)](https://hub.docker.com/r/memrook/oneway_subot)
[![License](https://img.shields.io/github/license/memrook/oneway_subot.svg?style=flat-square)](LICENSE)

Современный Telegram-бот для технической поддержки с расширенными возможностями, построенный на чистой архитектуре.

## 🚀 Возможности

### Основной функционал
- **Система тикетов**: Автоматическое создание и управление обращениями
- **Канал + Группа**: Интеграция канала для публикации и группы для обсуждения
- **Статусы тикетов**: Open → In Progress → Waiting → Closed → Rated
- **Система оценок**: 5-звездочная система с возможностью комментариев
- **Уведомления**: Автоматические уведомления о событиях

### Дополнительные возможности
- **Аналитика**: Подробная статистика и отчеты
- **Мониторинг**: Prometheus метрики и health checks
- **Кэширование**: Redis для повышения производительности
- **Логирование**: Структурированные логи с ротацией
- **Блокировка пользователей**: Система модерации
- **Многоязычность**: Поддержка русского языка

### Архитектурные улучшения
- **Clean Architecture**: Разделение на слои (Repository, Service, Handler)
- **Dependency Injection**: Слабосвязанные компоненты
- **Graceful Shutdown**: Корректное завершение работы
- **Health Checks**: Проверка состояния компонентов
- **Конфигурация**: YAML + переменные окружения

## 📁 Структура проекта

```
├── cmd/bot/              # Точка входа приложения
├── internal/
│   ├── config/           # Конфигурация
│   ├── handlers/         # Telegram handlers
│   ├── models/           # Модели данных
│   ├── repository/       # Слой доступа к данным
│   │   └── mongodb/      # MongoDB реализация
│   ├── service/          # Бизнес-логика
│   └── bot/              # Telegram bot setup
├── pkg/
│   ├── logger/           # Логирование
│   └── metrics/          # Метрики
├── configs/              # Файлы конфигурации
├── monitoring/           # Prometheus/Grafana
├── nginx/               # Reverse proxy
└── docker-compose.yml   # Docker окружение
```

## 🛠 Установка и запуск

### Требования
- Go 1.21+
- MongoDB 5.0+
- Redis 6.0+ (опционально)
- Docker & Docker Compose (для контейнерного запуска)

### 🚀 Быстрый запуск (рекомендуется)

1. **Клонирование и настройка**
```bash
git clone https://github.com/memrook/oneway_subot.git
cd oneway_subot
cp env.example .env
# Отредактируйте .env файл с вашими настройками
```

2. **Запуск одной командой**
```bash
./deploy.sh
# или
docker-compose up -d --build
```

3. **Проверка работы**
```bash
docker-compose ps
docker-compose logs -f bot
curl http://localhost:8081/health
```

### 🛠 Разработка

```bash
# Настройка среды разработки
make setup

# Локальный запуск с hot reload
make dev

# Тестирование
make test

# Проверка кода
make lint fmt
```

### 📦 Production развертывание

```bash
# Полное окружение с мониторингом
docker-compose --profile monitoring up -d

# С reverse proxy
docker-compose --profile proxy up -d
```

## ⚙️ Конфигурация

### Переменные окружения

| Переменная | Описание | Обязательная |
|-----------|----------|-------------|
| `BOT_TOKEN` | Токен Telegram бота | ✅ |
| `CHANNEL_ID` | ID канала для публикации тикетов | ✅ |
| `SUPERGROUP_ID` | ID группы для обсуждения | ✅ |
| `DATABASE_URI` | URI подключения к MongoDB | ✅ |
| `ADMIN_CHAT_ID` | ID чата администратора | ❌ |
| `REDIS_URI` | URI подключения к Redis | ❌ |
| `LOG_LEVEL` | Уровень логирования (debug, info, warn, error) | ❌ |

### Настройка Telegram

1. **Создание бота**
   - Создайте бота через [@BotFather](https://t.me/BotFather)
   - Получите токен и добавьте в `BOT_TOKEN`

2. **Настройка канала**
   - Создайте публичный канал
   - Добавьте бота как администратора
   - Получите ID канала и добавьте в `CHANNEL_ID`

3. **Настройка группы**
   - Создайте группу (обычную или супергруппу)
   - Добавьте бота как администратора
   - Свяжите группу с каналом как комментарии
   - Получите ID группы и добавьте в `SUPERGROUP_ID`

### Файл конфигурации

Основная конфигурация в `configs/config.yaml`:

```yaml
app:
  name: "OneWay Support Bot"
  version: "2.0.0"
  environment: "production"

bot:
  token: "${BOT_TOKEN}"
  channel_id: ${CHANNEL_ID}
  supergroup_id: ${SUPERGROUP_ID}
  rate_limit:
    requests_per_minute: 30

database:
  uri: "${DATABASE_URI}"
  name: "oneway_support"
  timeout: 30s
  max_pool_size: 100

logging:
  level: "info"
  format: "json"
  file: "logs/bot.log"

features:
  survey:
    enabled: true
    auto_send_after_close: true
    reminder_after: 1h
  notifications:
    enabled: true
  analytics:
    enabled: true
    daily_reports: true
```

## 📊 Мониторинг

### Метрики Prometheus

Доступны по адресу `http://localhost:9090/metrics`:

- `telegram_messages_total` - Общее количество сообщений
- `tickets_created_total` - Количество созданных тикетов
- `tickets_closed_total` - Количество закрытых тикетов
- `response_time_seconds` - Время ответа
- `user_satisfaction_rating` - Рейтинг удовлетворенности

### Health Checks

```bash
# Проверка состояния приложения
curl http://localhost:8081/health

# Проверка состояния базы данных
curl http://localhost:8081/health/db

# Проверка готовности
curl http://localhost:8081/ready
```

### Логи

Логи сохраняются в:
- `logs/bot.log` - основные логи приложения
- `logs/errors.log` - логи ошибок
- Docker logs: `docker-compose logs -f bot`

## 🔧 API

### Внутренние эндпоинты

- `GET /health` - Проверка состояния
- `GET /metrics` - Prometheus метрики
- `GET /api/stats` - Статистика тикетов
- `POST /api/tickets` - Создание тикета (внутренний)

### Webhook (опционально)

Для высоконагруженных систем можно настроить webhook:

```yaml
bot:
  webhook:
    enabled: true
    url: "https://your-domain.com/webhook"
    port: 8080
```

## 🧪 Тестирование

```bash
# Запуск всех тестов
go test ./...

# Тесты с покрытием
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Интеграционные тесты
go test -tags=integration ./...

# Бенчмарки
go test -bench=. ./...
```

## 🚀 Деплой

### Production через Docker

1. **Подготовка сервера**
```bash
# Установка Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh

# Установка Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

2. **Настройка переменных окружения**
```bash
# Создайте .env файл с production настройками
nano .env
```

3. **Запуск в production**
```bash
docker-compose -f docker-compose.yml --profile monitoring up -d
```

### Kubernetes (опционально)

```bash
# Применение манифестов
kubectl apply -f k8s/
```

## 🔒 Безопасность

### Рекомендации

1. **Переменные окружения**
   - Никогда не коммитьте `.env` файлы
   - Используйте секреты в production

2. **База данных**
   - Включите аутентификацию MongoDB
   - Используйте SSL/TLS соединения
   - Регулярно делайте бэкапы

3. **Сеть**
   - Используйте firewall
   - Ограничьте доступ к портам
   - Настройте reverse proxy

4. **Логи**
   - Не логируйте токены и пароли
   - Настройте ротацию логов
   - Мониторьте подозрительную активность

## 📈 Производительность

### Оптимизации

- **Connection pooling** для MongoDB
- **Redis кэширование** часто запрашиваемых данных
- **Batch operations** для массовых операций
- **Graceful degradation** при высокой нагрузке

### Масштабирование

- Горизонтальное масштабирование через Docker Swarm/Kubernetes
- Read replicas для MongoDB
- Load balancer для распределения нагрузки
- CDN для статических ресурсов

## 🐛 Отладка

### Частые проблемы

1. **Бот не отвечает**
```bash
# Проверьте логи
docker-compose logs bot

# Проверьте токен
curl https://api.telegram.org/bot<TOKEN>/getMe
```

2. **Ошибки базы данных**
```bash
# Проверьте подключение
docker-compose exec mongo mongosh

# Проверьте индексы
db.tickets.getIndexes()
```

3. **Высокая нагрузка**
```bash
# Мониторинг ресурсов
docker stats

# Проверка метрик
curl localhost:9090/metrics
```

## 🤝 Участие в разработке

1. Fork репозитория
2. Создайте feature branch (`git checkout -b feature/amazing-feature`)
3. Commit изменения (`git commit -m 'Add amazing feature'`)
4. Push в branch (`git push origin feature/amazing-feature`)
5. Создайте Pull Request

### Стандарты кода

- Используйте `gofmt` для форматирования
- Покрытие тестами не менее 80%
- Документируйте публичные функции
- Следуйте принципам SOLID

## 📄 Лицензия

Этот проект лицензирован под MIT License - см. файл [LICENSE](LICENSE) для деталей.

## 📞 Поддержка

- 🐛 [Issues](https://github.com/memrook/oneway_subot/issues)
- 💬 [Discussions](https://github.com/memrook/oneway_subot/discussions)
- 📧 Email: support@example.com

## 🏆 Благодарности

- [Telegram Bot API](https://core.telegram.org/bots/api)
- [MongoDB](https://www.mongodb.com/)
- [Go Community](https://golang.org/)

---

**OneWay Support Bot** - Современное решение для техподдержки в Telegram 🚀
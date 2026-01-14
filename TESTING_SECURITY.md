# Руководство по тестированию безопасности

Это руководство описывает различные способы тестирования исправлений безопасности в бэкенде Flowday.

## 📋 Содержание

1. [Unit тесты](#unit-тесты)
2. [Security тесты](#security-тесты)
3. [Ручное тестирование](#ручное-тестирование)
4. [Тестирование через API](#тестирование-через-api)

---

## Unit тесты

### Запуск тестов

```bash
# Запустить все тесты
go test ./...

# Запустить тесты с подробным выводом
go test -v ./...

# Запустить тесты для конкретного пакета
go test -v ./internal/middleware

# Запустить тесты с покрытием
go test -cover ./internal/middleware
```

### Доступные тесты

#### 1. JWT Middleware тесты (`internal/middleware/auth_test.go`)

- ✅ `TestAuthMiddleware_ValidToken` - Проверяет валидный токен
- ✅ `TestAuthMiddleware_InvalidAlgorithm` - Проверяет отклонение неправильного алгоритма
- ✅ `TestAuthMiddleware_MissingToken` - Проверяет отсутствие токена
- ✅ `TestAuthMiddleware_InvalidFormat` - Проверяет неправильный формат
- ✅ `TestAuthMiddleware_ExpiredToken` - Проверяет просроченный токен
- ✅ `TestAuthMiddleware_InvalidUserID` - Проверяет невалидный user_id

---

## Security тесты

### Автоматизированные security тесты

```bash
# Запустить security тесты (требуется запущенный сервер)
cd scripts
go run security_test.go
```

**Что тестируется:**

1. **JWT Algorithm Validation**
   - Проверка отклонения токенов с неправильным алгоритмом
   - Защита от algorithm confusion attacks

2. **File Upload Validation**
   - Отклонение слишком больших файлов (>5MB для аватаров, >10MB для вложений)
   - Отклонение файлов с опасными расширениями (.exe, .bat, .sh, и т.д.)
   - Защита от path traversal
   - Принятие валидных файлов

3. **Rate Limit Headers**
   - Проверка правильного формата заголовков
   - Убедиться, что числа конвертируются правильно (не как rune)

---

## Ручное тестирование

### 1. Тестирование JWT валидации

#### Тест 1: Валидный токен

```bash
# 1. Залогиниться и получить токен
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'

# 2. Использовать токен для защищенного endpoint
curl -X GET http://localhost:8080/api/v1/me \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Ожидаемый результат:** Статус 200, данные пользователя

#### Тест 2: Невалидный токен

```bash
curl -X GET http://localhost:8080/api/v1/me \
  -H "Authorization: Bearer invalid.token.here"
```

**Ожидаемый результат:** Статус 401, ошибка авторизации

#### Тест 3: Токен с неправильным алгоритмом (атака)

**Для этого теста нужно создать токен с alg=none:**
```python
# Python script для генерации токена с alg=none
import base64
import json
import time

header = {"alg": "none", "typ": "JWT"}
payload = {"user_id": "507f1f77bcf86cd799439011", "exp": int(time.time()) + 3600}

header_b64 = base64.urlsafe_b64encode(json.dumps(header).encode()).decode().rstrip('=')
payload_b64 = base64.urlsafe_b64encode(json.dumps(payload).encode()).decode().rstrip('=')

token = f"{header_b64}.{payload_b64}."

print(f"Token with alg=none: {token}")
```

**Ожидаемый результат:** Статус 401, токен отклонен (наша защита работает!)

---

### 2. Тестирование валидации файлов

#### Тест 1: Загрузка большого файла (аватар)

```bash
# Создать большой файл (6MB)
dd if=/dev/zero of=large.jpg bs=1M count=6

# Попытка загрузить
curl -X POST http://localhost:8080/api/v1/users/avatar \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "avatar=@large.jpg"
```

**Ожидаемый результат:** Статус 400, ошибка "File size exceeds maximum allowed size (5MB)"

#### Тест 2: Загрузка файла с опасным расширением

```bash
# Создать "вредный" файл
echo "malicious code" > test.exe

# Попытка загрузить как аватар
curl -X POST http://localhost:8080/api/v1/users/avatar \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "avatar=@test.exe"
```

**Ожидаемый результат:** Статус 400, ошибка "Invalid file type"

#### Тест 3: Path traversal атака

```bash
# Попытка использовать path traversal в имени файла
curl -X POST http://localhost:8080/api/v1/users/avatar \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "avatar=@test.png;filename=../../../etc/passwd.png"
```

**Ожидаемый результат:** Статус 400 или безопасное имя файла

#### Тест 4: Валидный файл

```bash
# Скачать тестовое изображение
curl -o test.png https://via.placeholder.com/100x100.png

# Загрузить
curl -X POST http://localhost:8080/api/v1/users/avatar \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "avatar=@test.png"
```

**Ожидаемый результат:** Статус 200, успешная загрузка

---

### 3. Тестирование Rate Limiting

#### Тест 1: Проверка заголовков rate limit

```bash
# Сделать запрос к endpoint с rate limiting
for i in {1..5}; do
  echo "Request $i:"
  curl -i -X POST http://localhost:8080/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d '{"email":"test@example.com","password":"wrong"}' | grep -i "X-RateLimit"
  sleep 1
done
```

**Ожидаемый результат:** 
- Заголовки `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` присутствуют
- Значения - числовые строки (не одиночные символы)

#### Тест 2: Превышение лимита

```bash
# Быстро отправить много запросов
for i in {1..10}; do
  curl -X POST http://localhost:8080/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d '{"email":"test@example.com","password":"wrong"}' &
done
wait
```

**Ожидаемый результат:** После нескольких запросов - статус 429 "Too many requests"

---

### 4. Тестирование через браузер (HTML)

Уже есть готовый HTML файл для тестирования rate limiting:

```bash
# Открыть в браузере
open scripts/test_rate_limit.html
# или
firefox scripts/test_rate_limit.html
```

---

## Тестирование через API

### Использование существующего скрипта

```bash
# Запустить сервер (в отдельном терминале)
go run server/main.go

# В другом терминале запустить тесты
cd scripts
go run verify_security.go
```

**Что тестирует скрипт:**
- Регистрация и логин
- Проверка refresh token cookie (HttpOnly, Secure)
- Проверка refresh flow
- Общие проверки безопасности

---

## Checklist для тестирования

### ✅ JWT Security
- [ ] Валидный токен работает
- [ ] Невалидный токен отклоняется
- [ ] Просроченный токен отклоняется
- [ ] Токен без "Bearer " префикса отклоняется
- [ ] Токен с неправильным алгоритмом отклоняется (защита от alg=none)

### ✅ File Upload Security
- [ ] Файлы больше 5MB (аватары) отклоняются
- [ ] Файлы больше 10MB (вложения) отклоняются
- [ ] Опасные расширения (.exe, .bat, .sh) отклоняются
- [ ] Path traversal попытки блокируются
- [ ] Валидные изображения принимаются

### ✅ Rate Limiting
- [ ] Rate limit заголовки правильно форматированы
- [ ] Превышение лимита возвращает 429
- [ ] Лимиты работают на auth endpoints

### ✅ Type Safety
- [ ] Нет паник при невалидных токенах
- [ ] Все type assertions безопасные

---

## Полезные команды

```bash
# Запустить все тесты
go test ./... -v

# Запустить тесты с покрытием
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Проверить код на ошибки
go vet ./...

# Проверить форматирование
gofmt -d .
```

---

## Дополнительные ресурсы

- [OWASP Testing Guide](https://owasp.org/www-project-web-security-testing-guide/)
- [JWT Security Best Practices](https://tools.ietf.org/html/rfc8725)
- [File Upload Security](https://owasp.org/www-community/vulnerabilities/Unrestricted_File_Upload)

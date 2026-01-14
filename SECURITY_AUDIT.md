# Отчет по анализу безопасности бэкенда Flowday

## Дата анализа: 2025-01-27

---

## 🔴 КРИТИЧЕСКИЕ УЯЗВИМОСТИ

### 1. Отсутствие валидации алгоритма JWT (CVE-подобная уязвимость)

**Файл:** `internal/middleware/auth.go:44-46`

**Проблема:**
```go
token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
    return getSecret(), nil  // ❌ Нет проверки алгоритма
})
```

**Риск:** Злоумышленник может использовать токен с алгоритмом `none` для обхода проверки подписи.

**Рекомендация:** Добавить проверку алгоритма:
```go
token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
    if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
        return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
    }
    return getSecret(), nil
})
```

---

### 2. Небезопасное приведение типов (Type Assertion Panic)

**Файл:** `internal/middleware/auth.go:55-56`

**Проблема:**
```go
claims := token.Claims.(jwt.MapClaims)
userIDStr := claims["user_id"].(string)  // ❌ Может вызвать panic
```

**Риск:** Если в токене отсутствует `user_id` или он имеет другой тип, приложение упадет с panic.

**Рекомендация:** Использовать безопасное приведение:
```go
claims, ok := token.Claims.(jwt.MapClaims)
if !ok {
    c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
    return
}
userIDStr, ok := claims["user_id"].(string)
if !ok {
    c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
    return
}
```

---

### 3. Хардкод секретных ключей

**Файл:** `internal/middleware/auth.go:15-20`, `internal/auth/jwt.go:11-24`

**Проблема:**
```go
func getSecret() []byte {
    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        return []byte("super-secret-key")  // ❌ Слабый дефолтный ключ
    }
    return []byte(secret)
}
```

**Риск:** 
- Слабый дефолтный ключ легко подобрать
- В production может быть использован дефолтный ключ

**Рекомендация:** 
- Требовать наличие переменной окружения в production
- Генерировать криптографически стойкие ключи
- Использовать секреты из vault в production

---

### 4. Небезопасная загрузка файлов

**Файлы:** `internal/handlers/user_handler.go:64-111`, `internal/handlers/message_handler.go:44-80`

**Проблемы:**
1. **Нет проверки типа файла:**
```go
file, err := c.FormFile("avatar")
ext := filepath.Ext(file.Filename)  // ❌ Полагается только на расширение
```

2. **Нет ограничения размера файла**
3. **Нет проверки MIME-типа**
4. **Риск Path Traversal:**
```go
filename := fmt.Sprintf("%s_%d%s", userID.Hex(), time.Now().Unix(), ext)
// ❌ Если ext содержит "../", возможен путь за пределы директории
```

5. **Статические файлы без аутентификации:**
```go
r.Static("/api/v1/uploads", "./uploads")  // ❌ Любой может скачать файлы
```

**Рекомендация:**
- Проверять MIME-тип файла (magic bytes)
- Ограничить размер файла (например, 5MB для аватаров)
- Санитизировать имена файлов
- Добавить middleware для защиты статических файлов
- Хранить файлы вне webroot или проверять доступ через API

---

### 5. Уязвимость в Rate Limiting

**Файл:** `internal/middleware/ratelimit.go:35-37`

**Проблема:**
```go
c.Header("X-RateLimit-Limit", string(rune(context.Limit)))  // ❌ Неправильное преобразование
c.Header("X-RateLimit-Remaining", string(rune(context.Remaining)))
c.Header("X-RateLimit-Reset", string(rune(context.Reset)))
```

**Риск:** 
- `string(rune(int))` неправильно преобразует числа в строки
- In-memory store не работает в распределенных системах

**Рекомендация:**
```go
c.Header("X-RateLimit-Limit", strconv.FormatInt(context.Limit, 10))
c.Header("X-RateLimit-Remaining", strconv.FormatInt(context.Remaining, 10))
c.Header("X-RateLimit-Reset", strconv.FormatInt(context.Reset, 10))
// Использовать Redis для production
```

---

## 🟠 ВЫСОКИЕ РИСКИ

### 6. Отсутствие Rate Limiting на критических endpoints

**Файл:** `internal/router/router.go:21-23`

**Проблема:**
```go
authGroup.POST("/reset-password", auth.ResetPasswordHandler)  // ❌ Нет rate limiting
authGroup.POST("/refresh", auth.RefreshHandler)  // ❌ Нет rate limiting
authGroup.POST("/logout", auth.LogoutHandler)  // ❌ Нет rate limiting
```

**Риск:** 
- Brute-force атаки на сброс пароля
- Злоупотребление refresh token endpoint

**Рекомендация:** Добавить rate limiting для всех auth endpoints.

---

### 7. Небезопасная конфигурация CORS

**Файл:** `server/main.go:40-51`

**Проблема:**
```go
AllowOriginFunc: func(origin string) bool {
    return strings.HasPrefix(origin, "http://localhost") ||
           strings.HasPrefix(origin, "http://127.0.0.1")  // ❌ Только для development
},
AllowCredentials: true,  // ⚠️ В сочетании с широким CORS - рискованно
```

**Риск:**
- В production должна быть строгая конфигурация
- `AllowCredentials: true` требует точного указания origins

**Рекомендация:**
- Использовать переменные окружения для allowed origins в production
- Валидировать origins через whitelist
- Использовать HTTPS в production

---

### 8. Информационное раскрытие через ошибки

**Файлы:** Различные handlers

**Проблемы:**
1. **Разные сообщения об ошибках:**
```go
// internal/auth/service.go:72
return "", "", errors.New("invalid credentials")  // ✅ Хорошо

// Но в других местах:
c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})  // ❌ Может раскрыть детали
```

2. **User Enumeration:**
```go
// internal/auth/service.go:99-106
func RequestPasswordReset(email string) error {
    // ...
    if err != nil {
        return nil  // ✅ Silent fail - хорошо
    }
```

**Рекомендация:**
- Унифицировать сообщения об ошибках
- Не раскрывать детали внутренней логики
- Использовать generic messages для пользователей

---

### 9. Отсутствие проверки алгоритма в JWT для refresh токенов

**Файл:** `internal/auth/jwt.go` (предположительно в ValidateRefreshToken)

**Проблема:** Аналогично проблеме #1, нужно проверить валидацию refresh токенов.

---

### 10. Недостаточная защита от NoSQL Injection

**Риск:** Хотя MongoDB driver предоставляет некоторую защиту, нужно убедиться что все запросы используют типизированные структуры.

**Рекомендация:** 
- Проверить все места где используется пользовательский ввод в BSON запросах
- Использовать prepared statements где возможно

---

## 🟡 СРЕДНИЕ РИСКИ

### 11. Cookie Security

**Файл:** `internal/auth/handler.go:63`

**Проблема:**
```go
c.SetCookie("refresh_token", refreshToken, int((time.Hour * 24 * 7).Seconds()), "/api/v1/auth", "", false, true)
//                                                                              ^^^ Secure flag = false
```

**Риск:** 
- `Secure: false` - cookie передается по HTTP
- В production должен быть `Secure: true` при использовании HTTPS

**Рекомендация:**
```go
secure := os.Getenv("ENV") == "production"
c.SetCookie("refresh_token", refreshToken, int((time.Hour * 24 * 7).Seconds()), "/api/v1/auth", "", secure, true)
```

---

### 12. Отсутствие проверки алгоритма подписи в Refresh Token

**Файл:** `internal/auth/jwt.go` (нужно проверить ValidateRefreshToken)

**Рекомендация:** Убедиться что ValidateRefreshToken проверяет алгоритм.

---

### 13. Логирование чувствительных данных

**Риск:** Проверить все места где используется `log.Printf` на наличие чувствительных данных.

**Рекомендация:** 
- Не логировать пароли, токены, PII
- Использовать структурированное логирование
- Маскировать чувствительные данные

---

### 14. Weak Password Requirements

**Файл:** `internal/auth/handler.go:15`

**Проблема:**
```go
Password string `json:"password" binding:"required,min=6"`  // ❌ Только минимум 6 символов
```

**Рекомендация:**
- Минимум 8-12 символов
- Требовать комбинацию букв, цифр и специальных символов
- Проверка на распространенные пароли

---

## ✅ ПОЛОЖИТЕЛЬНЫЕ МОМЕНТЫ

1. ✅ Использование bcrypt для хеширования паролей (cost = 14)
2. ✅ HTTP-Only cookies для refresh токенов
3. ✅ Использование JWT с разделением access/refresh токенов
4. ✅ Rate limiting на некоторых endpoints
5. ✅ Проверка доступа к проектам через `verifyProjectAccess`
6. ✅ Silent fail при запросе сброса пароля (защита от user enumeration)
7. ✅ Маркировка использованных кодов сброса пароля

---

## 📋 ПРИОРИТЕТНЫЙ ПЛАН ИСПРАВЛЕНИЙ

### Немедленно (Критично):
1. ✅ Добавить проверку алгоритма JWT
2. ✅ Исправить небезопасные type assertions
3. ✅ Добавить валидацию загружаемых файлов
4. ✅ Исправить rate limit headers
5. ✅ Удалить хардкод секретных ключей

### В ближайшее время (Высокий приоритет):
6. ✅ Добавить rate limiting на все auth endpoints
7. ✅ Настроить безопасный CORS для production
8. ✅ Защитить статические файлы
9. ✅ Настроить Secure флаг для cookies

### Постепенно (Средний приоритет):
10. ✅ Улучшить требования к паролям
11. ✅ Улучшить обработку ошибок
12. ✅ Аудит логирования
13. ✅ Добавить security headers (CSP, X-Frame-Options, etc.)

---

## 🔍 ДОПОЛНИТЕЛЬНЫЕ РЕКОМЕНДАЦИИ

1. **Добавить Security Headers:**
   - Content-Security-Policy
   - X-Frame-Options: DENY
   - X-Content-Type-Options: nosniff
   - Strict-Transport-Security (HSTS)

2. **Мониторинг и логирование:**
   - Логировать все попытки неудачной аутентификации
   - Алерты на подозрительную активность
   - Мониторинг rate limit violations

3. **Тестирование безопасности:**
   - Добавить unit тесты для middleware
   - Интеграционные тесты для auth flow
   - Penetration testing

4. **Документация:**
   - Документировать процесс rotation секретных ключей
   - Security best practices для команды

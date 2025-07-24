# Определяем тестовый USER_ID, который будет использоваться во всех запросах
USER_ID="a1b2c3d4-e5f6-7890-1234-567890abcdef"

# Имитируем User-Agent и IP-адрес для запросов.
# X-Forwarded-For используется для передачи IP-адреса, который будет прочитан приложением Fiber.
INITIAL_USER_AGENT="MyAwesomeClient/1.0"
INITIAL_IP_ADDRESS="192.168.1.10"

echo "--- Шаг 1: Генерация новой пары токенов ---"
# POST /api/v1/auth/token
# Генерируем access и refresh токены для USER_ID.
RESPONSE=$(curl -s -X POST \
  "http://localhost:8000/api/v1/auth/token?user_id=${USER_ID}" \
  -H "Content-Type: application/json" \
  -H "User-Agent: ${INITIAL_USER_AGENT}" \
  -H "X-Forwarded-For: ${INITIAL_IP_ADDRESS}")

ACCESS_TOKEN=$(echo "${RESPONSE}" | jq -r .access_token)
REFRESH_TOKEN=$(echo "${RESPONSE}" | jq -r .refresh_token)

echo "  Получен access token: ${ACCESS_TOKEN}"
echo "  Получен refresh token: ${REFRESH_TOKEN}"
echo ""


echo "--- Шаг 2: Получение GUID текущего пользователя (успешное) ---"
# GET /api/v1/user/me
# Используем полученный ACCESS_TOKEN для проверки аутентификации.
curl -s -X GET \
  "http://localhost:8000/api/v1/user/me" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}"
echo ""
echo ""

echo "--- Шаг 3: Обновление пары токенов (успешное) ---"
# POST /api/v1/auth/token/refresh
# Используем ACCESS_TOKEN и REFRESH_TOKEN для получения новой пары.
# User-Agent и IP те же, что и при генерации, поэтому ожидаем успех.
NEW_USER_AGENT_SUCCESS="${INITIAL_USER_AGENT}"
NEW_IP_ADDRESS_SUCCESS="${INITIAL_IP_ADDRESS}"

REFRESH_RESPONSE=$(curl -s -X POST \
  "http://localhost:8000/api/v1/auth/token/refresh" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -H "User-Agent: ${NEW_USER_AGENT_SUCCESS}" \
  -H "X-Forwarded-For: ${NEW_IP_ADDRESS_SUCCESS}" \
  -d "{
        \"refresh_token\": \"${REFRESH_TOKEN}\"
      }")

echo ${REFRESH_RESPONSE}
NEW_ACCESS_TOKEN=$(echo "${REFRESH_RESPONSE}" | jq -r .access_token)
NEW_REFRESH_TOKEN=$(echo "${REFRESH_RESPONSE}" | jq -r .refresh_token)

echo "  Новый access token: ${NEW_ACCESS_TOKEN}"
echo "  Новый refresh token: ${NEW_REFRESH_TOKEN}"
echo ""

echo "--- Шаг 3.1: Проверка старых токенов после обновления ---"
# Старый refresh токен должен быть отозван. Старый access токен перестает быть частью валидной пары.
echo "  Попытка использовать старый access token для получения GUID:"
curl -s -X GET \
  "http://localhost:8000/api/v1/user/me" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" # Используем СТАРЫЙ ACCESS_TOKEN
echo ""

echo "  Попытка использовать старый refresh token для обновления:"
curl -s -X POST \
  "http://localhost:8000/api/v1/auth/token/refresh" \
  -H "Authorization: Bearer ${NEW_ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -H "User-Agent: ${NEW_USER_AGENT_SUCCESS}" \
  -H "X-Forwarded-For: ${NEW_IP_ADDRESS_SUCCESS}" \
  -d "{
        \"refresh_token\": \"${REFRESH_TOKEN}\"
      }" # Используем СТАРЫЙ REFRESH_TOKEN
echo ""
echo ""

echo "--- Шаг 4: Обновление пары токенов (с новым IP-адресом - уведомление) ---"
# POST /api/v1/auth/token/refresh
# Используем последнюю полученную пару токенов, но с другим IP-адресом.
# Ожидаем новую пару токенов, и в консоли Docker Compose будут логи о вебхуке.
DIFFERENT_IP_ADDRESS="172.16.0.200" # Имитируем новый IP-адрес

IP_CHANGE_REFRESH_RESPONSE=$(curl -s -X POST \
  "http://localhost:8000/api/v1/auth/token/refresh" \
  -H "Authorization: Bearer ${NEW_ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -H "User-Agent: ${NEW_USER_AGENT_SUCCESS}" \
  -H "X-Forwarded-For: ${DIFFERENT_IP_ADDRESS}" \
  -d "{
        \"refresh_token\": \"${NEW_REFRESH_TOKEN}\"
    }")

NEW_ACCESS_TOKEN_AFTER_IP=$(echo "${IP_CHANGE_REFRESH_RESPONSE}" | jq -r .access_token)
NEW_REFRESH_TOKEN_AFTER_IP=$(echo "${IP_CHANGE_REFRESH_RESPONSE}" | jq -r .refresh_token)

echo "  Новый access token после смены IP: ${NEW_ACCESS_TOKEN_AFTER_IP}"
echo "  Новый refresh token после смены IP: ${NEW_REFRESH_TOKEN_AFTER_IP}"
echo "  (Проверьте логи Docker Compose для уведомления о новом IP)"
echo ""

echo "--- Шаг 5: Обновление пары токенов (с другим User-Agent - ошибка и деавторизация) ---"
# POST /api/v1/auth/token/refresh
# Используем последнюю полученную пару токенов, но с другим User-Agent.
# Ожидаем ошибку 401, и все токены пользователя будут отозваны.
DIFFERENT_USER_AGENT="MaliciousBot/1.0"

curl -s -X POST \
  "http://localhost:8000/api/v1/auth/token/refresh" \
  -H "Authorization: Bearer ${NEW_ACCESS_TOKEN_AFTER_IP}" \
  -H "Content-Type: application/json" \
  -H "User-Agent: ${DIFFERENT_USER_AGENT}" \
  -H "X-Forwarded-For: ${DIFFERENT_IP_ADDRESS}" \
  -d "{
        \"refresh_token\": \"${NEW_REFRESH_TOKEN_AFTER_IP}\"
      }"
echo ""

echo "--- Шаг 5.1: Проверка доступа после смены User-Agent (ожидаем ошибку) ---"
# GET /api/v1/user/me
# После попытки смены User-Agent, все токены пользователя должны быть отозваны.
echo "  Попытка использовать access token после ошибки User-Agent:"
curl -s -X GET \
  "http://localhost:8000/api/v1/user/me" \
  -H "Authorization: Bearer ${NEW_ACCESS_TOKEN_AFTER_IP}"
echo ""
echo ""

echo "--- Шаг 6: Деавторизация пользователя (Logout) ---"
# POST /api/v1/auth/token/logout
# Для этого шага, так как предыдущая попытка смены User-Agent отозвала все токены,
# нам нужно сначала сгенерировать новую пару токенов для того же пользователя.
echo "  Генерируем новую пару токенов для деавторизации:"
RESPONSE_FOR_LOGOUT=$(curl -s -X POST \
  "http://localhost:8000/api/v1/auth/token?user_id=${USER_ID}" \
  -H "Content-Type: application/json" \
  -H "User-Agent: ${INITIAL_USER_AGENT}" \
  -H "X-Forwarded-For: ${INITIAL_IP_ADDRESS}")

ACCESS_TOKEN_FOR_LOGOUT=$(echo "${RESPONSE_FOR_LOGOUT}" | jq -r .access_token)
REFRESH_TOKEN_FOR_LOGOUT=$(echo "${RESPONSE_FOR_LOGOUT}" | jq -r .refresh_token)

echo "  Новый access token для логаута: ${ACCESS_TOKEN_FOR_LOGOUT}"
echo "  Новый refresh token для логаута: ${REFRESH_TOKEN_FOR_LOGOUT}"
echo ""

echo "  Выполняем запрос на деавторизацию:"
curl -s -X POST \
  "http://localhost:8000/api/v1/auth/token/logout" \
  -H "Authorization: Bearer ${ACCESS_TOKEN_FOR_LOGOUT}"
echo ""
echo ""

echo "--- Шаг 6.1: Проверка деавторизации (ожидаем ошибку) ---"
# GET /api/v1/user/me
# После логаута, access токен больше не должен быть валидным.
echo "  Попытка использовать access token после деавторизации:"
curl -s -X GET \
  "http://localhost:8000/api/v1/user/me" \
  -H "Authorization: Bearer ${ACCESS_TOKEN_FOR_LOGOUT}"
echo ""

echo "  Попытка использовать refresh token после деавторизации:"
curl -s -X POST \
  "http://localhost:8000/api/v1/auth/token/refresh" \
  -H "Authorization: Bearer ${ACCESS_TOKEN_FOR_LOGOUT}" \
  -H "Content-Type: application/json" \
  -H "User-Agent: ${INITIAL_USER_AGENT}" \
  -H "X-Forwarded-For: ${INITIAL_IP_ADDRESS}" \
  -d "{
        \"refresh_token\": \"${REFRESH_TOKEN_FOR_LOGOUT}\"
      }"
echo ""

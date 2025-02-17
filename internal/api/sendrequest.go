package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
	"trading/config"
)

func generateSignature(secret string, params map[string]string) string {
	queryString := ""
	for key, value := range params {
		queryString += fmt.Sprintf("%s=%s&", key, value)
	}

	if len(queryString) > 0 {
		queryString = queryString[:len(queryString)-1] // Убираем последний '&', если строка не пустая
	}

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(queryString))
	return hex.EncodeToString(h.Sum(nil))
}

// SendRequest - Функция для отправки HTTP-запроса
func SendRequest(endpoint, method string, params map[string]string, signed bool) ([]byte, error) {
	if params == nil {
		params = map[string]string{} // Инициализация пустой карты, если params == nil
	}

	client := &http.Client{}
	url := fmt.Sprintf("%s%s", config.BaseURL, endpoint)

	// Добавляем параметры запроса для GET
	queryString := ""
	if method == "GET" && len(params) > 0 {
		queryString = "?"
		for key, value := range params {
			queryString += fmt.Sprintf("%s=%s&", key, value)
		}
		queryString = queryString[:len(queryString)-1]
		url += queryString
	}

	// Создаем HTTP-запрос
	var req *http.Request
	var err error
	if method == "POST" {
		jsonBody, _ := json.Marshal(params)
		req, err = http.NewRequest("POST", url, bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest("GET", url, nil)
	}
	if err != nil {
		return nil, err
	}

	// Устанавливаем заголовки
	req.Header.Set("X-BM-KEY", config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	if signed {
		req.Header.Set("X-BM-SIGN", generateSignature(config.APISecret, params))
		req.Header.Set("X-BM-TIMESTAMP", strconv.FormatInt(time.Now().UnixMilli(), 10))
	}

	// Отправляем запрос
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

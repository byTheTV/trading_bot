package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"main/config" // Локальный импорт, проверьте структуру проекта
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

// Функция для отправки HTTP-запроса
func sendRequest(endpoint, method string, params map[string]string, signed bool) ([]byte, error) {
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

// Функция для получения данных стакана по торговой паре
func getOrderBook(symbol string) (float64, float64, error) {
	endpoint := fmt.Sprintf("/spot/v1/symbols/book?symbol=%s&limit=5", symbol)

	// Отправляем запрос
	resp, err := sendRequest(endpoint, "GET", nil, false)
	if err != nil {
		return 0, 0, err
	}

	// Декодируем JSON
	var result map[string]interface{}
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return 0, 0, fmt.Errorf("Ошибка парсинга JSON: %v", err)
	}

	// Проверяем, содержит ли ответ данные
	if code, ok := result["code"].(float64); !ok || code != 1000 {
		return 0, 0, fmt.Errorf("Ошибка запроса стакана для пары %s: %v", symbol, result["message"])
	}

	// Получаем первую цену покупки и продажи
	data := result["data"].(map[string]interface{})
	var buyPrice, sellPrice float64
	if buys, ok := data["buys"].([]interface{}); ok && len(buys) > 0 {
		buyPrice, _ = strconv.ParseFloat(buys[0].(map[string]interface{})["price"].(string), 64)
	}
	if sells, ok := data["sells"].([]interface{}); ok && len(sells) > 0 {
		sellPrice, _ = strconv.ParseFloat(sells[0].(map[string]interface{})["price"].(string), 64)
	}

	return buyPrice, sellPrice, nil
}

func getBalance() (map[string]float64, error) {
	endpoint := "/v1/account/balances"
	resp, err := sendRequest(endpoint, "GET", map[string]string{}, true) // Передаём пустую карту
	if err != nil {
		return nil, err
	}

	// Декодируем JSON
	var result map[string]interface{}
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return nil, fmt.Errorf("Ошибка парсинга JSON: %v", err)
	}

	// Собираем баланс
	balances := make(map[string]float64)
	if data, ok := result["data"].(map[string]interface{}); ok {
		if wallet, ok := data["wallet"].([]interface{}); ok {
			for _, item := range wallet {
				balanceData := item.(map[string]interface{})
				currency := balanceData["currency"].(string)
				available, _ := strconv.ParseFloat(balanceData["available"].(string), 64)
				balances[currency] = available
			}
		}
	}
	return balances, nil
}

// Обновлённая функция placeOrder для возврата ID ордера
func placeOrder(symbol, side, orderType string, size, price float64) (string, error) {
	endpoint := "/spot/v1/submit_order"
	params := map[string]string{
		"symbol":    symbol,
		"side":      side,
		"type":      orderType,
		"size":      fmt.Sprintf("%.4f", size),
		"price":     fmt.Sprintf("%.2f", price),
		"timestamp": strconv.FormatInt(time.Now().UnixMilli(), 10),
	}

	resp, err := sendRequest(endpoint, "POST", params, true)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return "", fmt.Errorf("Ошибка парсинга JSON при размещении ордера: %v", err)
	}

	if code, ok := result["code"].(float64); !ok || code != 1000 {
		return "", fmt.Errorf("Ошибка размещения ордера: %v", result["message"])
	}

	data := result["data"].(map[string]interface{})
	orderID := data["order_id"].(string)
	return orderID, nil
}

func readPairsFromFile(filename string) ([]string, error) {
	// Считываем содержимое файла
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла %s: %v", filename, err)
	}

	// Разделяем строки на массив пар и убираем лишние пробелы
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var pairs []string
	for _, line := range lines {
		pair := strings.TrimSpace(line)
		if pair != "" { // Убираем пустые строки
			pairs = append(pairs, pair)
		}
	}

	if len(pairs) == 0 {
		return nil, fmt.Errorf("файл %s пуст или не содержит пар", filename)
	}

	return pairs, nil
}

// Структура для отслеживания ордеров
type Order struct {
	ID    string
	Price float64
	Size  float64
	Side  string // "buy" или "sell"
}

// Глобальная карта для хранения активных ордеров
var existingOrders = make(map[string]*Order)

// Загружаем пары из файла
func loadPairs(filename string) ([]string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("Ошибка чтения файла %s: %v", filename, err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var pairs []string
	for _, line := range lines {
		pair := strings.TrimSpace(line)
		if pair != "" {
			pairs = append(pairs, pair)
		}
	}

	if len(pairs) == 0 {
		return nil, fmt.Errorf("Файл %s пуст или не содержит пар", filename)
	}

	return pairs, nil
}

// Обработка ордера на продажу
func handleSellOrder(pair string, balance, sellPrice float64) {
	adjustedSellPrice := sellPrice * 0.999
	existingOrder, exists := existingOrders[pair+"_sell"]
	if exists && existingOrder.Price > adjustedSellPrice {
		fmt.Printf("Наш ордер на продажу перебит. Удаляем старый ордер и размещаем новый.\n")
		cancelOrder(existingOrder.ID)
	}

	fmt.Printf("Выставляем ордер на продажу %.4f %s по цене %.2f USD\n", balance, pair, adjustedSellPrice)
	orderID, err := placeOrder(pair, "sell", "limit", balance, adjustedSellPrice)
	if err != nil {
		fmt.Printf("Ошибка размещения ордера на продажу: %v\n", err)
	} else {
		existingOrders[pair+"_sell"] = &Order{ID: orderID, Price: adjustedSellPrice, Size: balance, Side: "sell"}
	}
}

// Обработка ордера на покупку
func handleBuyOrder(pair string, buyPrice, targetValue float64) {
	adjustedBuyPrice := buyPrice * 1.0001
	size := targetValue / adjustedBuyPrice

	existingOrder, exists := existingOrders[pair+"_buy"]
	if exists {
		fmt.Printf("Ордер на покупку уже существует для пары %s по цене %.2f. Новый ордер не создаётся.\n", pair, existingOrder.Price)
		return
	}

	fmt.Printf("Размещаем ордер на покупку %.4f %s по цене %.2f USD\n", size, pair, adjustedBuyPrice)
	orderID, err := placeOrder(pair, "buy", "limit", size, adjustedBuyPrice)
	if err != nil {
		fmt.Printf("Ошибка размещения ордера на покупку: %v\n", err)
	} else {
		existingOrders[pair+"_buy"] = &Order{ID: orderID, Price: adjustedBuyPrice, Size: size, Side: "buy"}
	}
}

// Отменяет ордер на покупку, если он существует
func cancelBuyOrderIfExists(pair string) {
	existingOrder, exists := existingOrders[pair+"_buy"]
	if exists {
		fmt.Printf("Отменяем ордер на покупку для пары %s, так как разница меньше 1%%.\n", pair)
		cancelOrder(existingOrder.ID)
		delete(existingOrders, pair+"_buy")
	}
}

// Обрабатывает отдельную пару: покупка/продажа, проверка ордеров
func processPair(pair string, balances map[string]float64, targetValue float64) {
	buyPrice, sellPrice, err := getOrderBook(pair)
	if err != nil {
		fmt.Printf("Ошибка получения стакана для пары %s: %v\n", pair, err)
		return
	}

	baseCurrency := strings.Split(pair, "_")[0]
	balance := balances[baseCurrency]
	fmt.Printf("Баланс для %s: %.4f %s (эквивалент %.2f USD)\n", baseCurrency, balance, baseCurrency, balance*sellPrice)

	// Обработка ордеров на продажу
	if balance > 0 && balance*sellPrice > 1.0 {
		handleSellOrder(pair, balance, sellPrice)
		return
	}

	// Обработка ордеров на покупку
	diff := ((sellPrice - buyPrice) / buyPrice) * 100
	fmt.Printf("Разница между ценой покупки и продажи: %.2f%%\n", diff)

	if diff > 1.0 {
		handleBuyOrder(pair, buyPrice, targetValue)
	} else {
		cancelBuyOrderIfExists(pair)
	}
}

// Функция для постоянной проверки ордеров
func monitorPairs(pairs []string, targetValue float64) {
	for {
		// Получаем текущий баланс
		balances, err := getBalance()
		if err != nil {
			fmt.Printf("Ошибка получения баланса: %v\n", err)
			time.Sleep(10 * time.Second) // Ждём перед следующей попыткой
			continue
		}

		// Обрабатываем каждую пару
		for _, pair := range pairs {
			fmt.Printf("Обрабатываем пару: %s\n", pair)
			processPair(pair, balances, targetValue)
		}

		fmt.Println("Ожидание перед следующим циклом...")
		time.Sleep(3 * time.Second) // Задержка перед следующим циклом
	}
}

// Функция для отмены ордера
func cancelOrder(orderID string) {
	endpoint := fmt.Sprintf("/spot/v1/cancel_order?order_id=%s", orderID)
	_, err := sendRequest(endpoint, "POST", nil, true)
	if err != nil {
		fmt.Printf("Ошибка отмены ордера с ID %s: %v\n", orderID, err)
	} else {
		fmt.Printf("Ордер с ID %s успешно отменён.\n", orderID)
	}
}

func main() {
	inputFile := "pairs.txt"
	targetValue := 20.0

	// Загружаем пары из файла
	pairs, err := loadPairs(inputFile)
	if err != nil {
		fmt.Printf("Ошибка загрузки пар: %v\n", err)
		return
	}

	fmt.Printf("Загружены пары: %v\n", pairs)

	// Запускаем мониторинг пар
	monitorPairs(pairs, targetValue)
}

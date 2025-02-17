package api

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Функция для получения данных стакана по торговой паре
func GetOrderBook(symbol string) (float64, float64, error) {
	endpoint := fmt.Sprintf("/spot/v1/symbols/book?symbol=%s&limit=5", symbol)

	// Отправляем запрос
	resp, err := SendRequest(endpoint, "GET", nil, false)
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

func GetBalance() (map[string]float64, error) {
	endpoint := "/v1/account/balances"
	resp, err := SendRequest(endpoint, "GET", map[string]string{}, true) // Передаём пустую карту
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

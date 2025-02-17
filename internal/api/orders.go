package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Обновлённая функция placeOrder для возврата ID ордера
func PlaceOrder(symbol, side, orderType string, size, price float64) (string, error) {
	endpoint := "/spot/v1/submit_order"
	params := map[string]string{
		"symbol":    symbol,
		"side":      side,
		"type":      orderType,
		"size":      fmt.Sprintf("%.4f", size),
		"price":     fmt.Sprintf("%.2f", price),
		"timestamp": strconv.FormatInt(time.Now().UnixMilli(), 10),
	}

	resp, err := SendRequest(endpoint, "POST", params, true)
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

func CancelOrder(orderID string) {
	endpoint := fmt.Sprintf("/spot/v1/cancel_order?order_id=%s", orderID)
	_, err := SendRequest(endpoint, "POST", nil, true)
	if err != nil {
		fmt.Printf("Ошибка отмены ордера с ID %s: %v\n", orderID, err)
	} else {
		fmt.Printf("Ордер с ID %s успешно отменён.\n", orderID)
	}
}

package trading

import (
	"fmt"
	"strings"
	"time"
	"trading/internal/api"
	"trading/internal/models"
)

// Глобальная карта для хранения активных ордеров
var existingOrders = make(map[string]*models.Order)

// Обработка ордера на продажу
func handleSellOrder(pair string, balance, sellPrice float64) {
	adjustedSellPrice := sellPrice * 0.999
	existingOrder, exists := existingOrders[pair+"_sell"]
	if exists && existingOrder.Price > adjustedSellPrice {
		fmt.Printf("Наш ордер на продажу перебит. Удаляем старый ордер и размещаем новый.\n")
		api.CancelOrder(existingOrder.ID)
	}

	fmt.Printf("Выставляем ордер на продажу %.4f %s по цене %.2f USD\n", balance, pair, adjustedSellPrice)
	orderID, err := api.PlaceOrder(pair, "sell", "limit", balance, adjustedSellPrice)
	if err != nil {
		fmt.Printf("Ошибка размещения ордера на продажу: %v\n", err)
	} else {
		existingOrders[pair+"_sell"] = &models.Order{ID: orderID, Price: adjustedSellPrice, Size: balance, Side: "sell"}
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
	orderID, err := api.PlaceOrder(pair, "buy", "limit", size, adjustedBuyPrice)
	if err != nil {
		fmt.Printf("Ошибка размещения ордера на покупку: %v\n", err)
	} else {
		existingOrders[pair+"_buy"] = &models.Order{ID: orderID, Price: adjustedBuyPrice, Size: size, Side: "buy"}
	}
}

// Отменяет ордер на покупку, если он существует
func cancelBuyOrderIfExists(pair string) {
	existingOrder, exists := existingOrders[pair+"_buy"]
	if exists {
		fmt.Printf("Отменяем ордер на покупку для пары %s, так как разница меньше 1%%.\n", pair)
		api.CancelOrder(existingOrder.ID)
		delete(existingOrders, pair+"_buy")
	}
}

// Обрабатывает отдельную пару: покупка/продажа, проверка ордеров
func processPair(pair string, balances map[string]float64, targetValue float64) {
	buyPrice, sellPrice, err := api.GetOrderBook(pair)
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
func MonitorPairs(pairs []string, targetValue float64) {
	for {
		// Получаем текущий баланс
		balances, err := api.GetBalance()
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

package main

import (
	"fmt"
	"trading/internal/trading"
	"trading/internal/utils"
)

func main() {
	inputFile := "pairs.txt"
	targetValue := 20.0

	// Загружаем пары из файла
	pairs, err := utils.LoadPairs(inputFile)
	if err != nil {
		fmt.Printf("Ошибка загрузки пар: %v\n", err)
		return
	}

	fmt.Printf("Загружены пары: %v\n", pairs)

	// Запускаем мониторинг пар
	trading.MonitorPairs(pairs, targetValue)
}

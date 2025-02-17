package models

// Order представляет структуру ордера
type Order struct {
	ID    string
	Price float64
	Size  float64
	Side  string // "buy" или "sell"
}

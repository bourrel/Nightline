package svcapi

// Buyers model
type Buyers struct {
	UserID int64   `json:"id"`
	Amount float64 `json:"amount"`
}

// Drinks model
type Drinks struct {
	DrinkID int64   `json:"id"`
	Amount  float64 `json:"amount"`
}

// ShoppingBasket model
type ShoppingBasket struct {
	Buyers []Buyers `json:"buyers"`
	Drinks []Drinks `json:"drinks"`
}

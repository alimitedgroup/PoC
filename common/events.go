package common

type AddStockEvent struct {
	MerceId int64 `json:"merce_id"`
	Stock   int64 `json:"stock"`
}

type MerceStock struct {
	MerceId int64 `json:"merce_id"`
	Stock   int64 `json:"stock"`
}

type CreateOrderEvent struct {
	OrderId int64        `json:"order_id"`
	Note    string       `json:"note"`
	Merci   []MerceStock `json:"merci"`
}

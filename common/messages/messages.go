package messages

import "github.com/google/uuid"

type CatalogItem struct {
	Id   uint64 `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

type CreateCatalogItem struct {
	Name string `json:"name" required:"true"`
}

type GetCatalogItem struct {
	Id uint64 `json:"id"`
}

type StockUpdate []struct {
	GoodId uint64 `json:"good_id"`
	Amount int    `json:"amount"`
}

type Reservation struct {
	ID            uuid.UUID `json:"id"`
	ReservedStock []struct {
		GoodId uint64 `json:"good_id"`
		Amount int    `json:"amount"`
	} `json:"reserved_stock"`
}

type ReserveStock struct {
	ID             uuid.UUID `json:"id"`
	RequestedStock []struct {
		GoodId uint64 `json:"good_id"`
		Amount int    `json:"amount"`
	} `json:"requested_stock"`
}

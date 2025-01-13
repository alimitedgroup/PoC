package messages

import "github.com/google/uuid"

type CatalogItem struct {
	Id   string `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

type CreateCatalogItem struct {
	Name string `json:"name" required:"true"`
}

type GetCatalogItem struct {
	Id string `json:"id"`
}

type StockUpdate []struct {
	GoodId string `json:"good_id"`
	Amount int    `json:"amount"`
}

type Reservation struct {
	ID            uuid.UUID `json:"id"`
	ReservedStock []struct {
		GoodId string `json:"good_id"`
		Amount int    `json:"amount"`
	} `json:"reserved_stock"`
}

type ReserveStock struct {
	ID             uuid.UUID `json:"id"`
	RequestedStock []struct {
		GoodId string `json:"good_id"`
		Amount int    `json:"amount"`
	} `json:"requested_stock"`
}

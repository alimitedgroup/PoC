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

type StockUpdate []StockUpdateItem

type StockUpdateItem struct {
	GoodId string `json:"good_id"`
	Amount int    `json:"amount"`
}

type Reservation struct {
	ID            uuid.UUID         `json:"id"`
	ReservedStock []ReservationItem `json:"reserved_stock"`
}

type ReservationItem struct {
	GoodId string `json:"good_id"`
	Amount int    `json:"amount"`
}

type ReserveStock struct {
	ID             uuid.UUID          `json:"id"`
	RequestedStock []ReserveStockItem `json:"requested_stock"`
}

type ReserveStockItem struct {
	GoodId string `json:"good_id"`
	Amount int    `json:"amount"`
}

type CreateOrder struct {
	Items []struct {
		GoodId string `json:"good_id"`
		Amount int    `json:"amount"`
	} `json:"items"`
}

type OrderCreated struct {
	ID         uuid.UUID              `json:"id"`
	Warehouses []OrderCreateWarehouse `json:"warehouses"`
}

type OrderCreateWarehouse struct {
	WarehouseId   string             `json:"warehouse_id"`
	ReservationId uuid.UUID          `json:"reservation_id"`
	Parts         []OrderCreatedItem `json:"parts"`
}

type OrderCreatedItem struct {
	GoodId string `json:"good_id"`
	Amount int    `json:"amount"`
}

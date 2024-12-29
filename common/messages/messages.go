package messages

type CatalogItem struct {
	Id   uint64 `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

type GetCatalogItem struct {
	Id uint64 `json:"id"`
}

type StockUpdate []struct {
	GoodId uint64 `json:"good_id"`
	Amount int    `json:"amount"`
}

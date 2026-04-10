package response

import "time"

type ProductDtoResponse struct {
	ID              uint    `json:"id"`
	SKU             string  `json:"sku"`
	NamaProduk      string  `json:"nama_produk"`
	Satuan          string  `json:"satuan"`
	HargaBeli       float64 `json:"harga_beli"`
	HargaJualTunai  float64 `json:"harga_jual_tunai"`
	HargaJualHutang float64 `json:"harga_jual_hutang"`
	Stok            int     `json:"stok"`

	UpdatedAt time.Time `json:"updated_at"`
}

type ProductAddBulkResponse struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

type GetProductResponse struct {
	UserId   uint               `json:"user_id"`
	NameUser uint               `json:"name_user"`
	Product  ProductDtoResponse `json:"product"`
}

type GetAllProductResponse struct {
	UserId      uint                 `json:"user_id"`
	NameUser    uint                 `json:"name_user"`
	HashNext    bool                 `json:"hash_next"`
	NextCursor  uint                 `json:"next_cursor"`
	ProductList []ProductDtoResponse `json:"product_list"`
}

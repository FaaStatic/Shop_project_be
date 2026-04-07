


type SearchProduct struct {
	ID         uint   `json:"id,omitempty"`
	Sku        string `json:"id,omitempty"`
	NamaProduk string `json:"id,omitempty"`
}

type GetProduct struct {
	ID uint `json:"id" validate:"required"`
}

type AddOneProduct struct {
	UserId          uint    `json:"user_id" validate:"required"`
	SKU             string  `json:"sku" validate="required"`
	NamaProduk      string  `json:"nama_produk" validate="required"`
	Satuan          string  `json:"satuan" validate:"required,oneof=pcs kg liter kardus ikat"`
	HargaBeli       float64 `json:"harga_beli" validate="required"`
	HargaJualTunai  float64 `json:"harga_jual_tunai" validate="required"`
	HargaJualHutang float64 `json:"harga_jual_hutang" validate="required"`
	Stok            int     `json:"stok" validate="required"`
}

type AddBulkProduct struct {
}

type DeleteProduct struct {
}

type UpdateProduct struct {
}

type GetAllProduct struct {
}
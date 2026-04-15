package request

import "mime/multipart"

type SearchProduct struct {
	ID         string `query:"id,omitempty"`
	Sku        string `query:"id,omitempty"`
	NamaProduk string `query:"id,omitempty"`
}

type GetProduct struct {
	ID string `query:"id" validate:"required"`
}

type AddProduct struct {
	UserId          string  `json:"user_id" validate:"required"`
	SKU             string  `json:"sku" validate="required"`
	NamaProduk      string  `json:"nama_produk" validate="required"`
	Satuan          string  `json:"satuan" validate:"required,oneof=pcs kg liter kardus ikat"`
	HargaBeli       float64 `json:"harga_beli" validate="required"`
	HargaJualTunai  float64 `json:"harga_jual_tunai" validate="required"`
	HargaJualHutang float64 `json:"harga_jual_hutang" validate="required"`
	Stok            int     `json:"stok" validate="required"`
}

type AddBulkProduct struct {
	UserId     string                `form:"user_id" validate:"required"`
	NameFile   string                `form:"name_file"`
	FileUpload *multipart.FileHeader `form:"file_upload"`
}

type DeleteProduct struct {
	ID uint `json:"id" validate:"required"`
}

type UpdateProduct struct {
	ID              string  `json:"id" validate:"required"`
	SKU             string  `json:"sku,omitempty"`
	NamaProduk      string  `json:"nama_produk,omitempty"`
	Satuan          string  `json:"satuan,omitempty" validate:"omitempty,oneof=pcs kg liter kardus ikat"`
	HargaBeli       float64 `json:"harga_beli,omitempty"`
	HargaJualTunai  float64 `json:"harga_jual_tunai,omitempty"`
	HargaJualHutang float64 `json:"harga_jual_hutang,omitempty"`
	Stok            int     `json:"stok,omitempty"`
}

type GetAllProduct struct {
	UserId string `json:"user_id" validate:"required"`
	Page   int    `query:"page"`
	Limit  int    `query:"limit"`
	Search string `query:"search"`
}

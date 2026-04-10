package domain

import (
	"context"
	"shop_project_be/internal/dto/request"
	"shop_project_be/internal/dto/response"
	"time"

	"gorm.io/gorm"
)

type Products struct {
	ID              uint    `gorm:"primaryKey" json:"id"`
	SKU             string  `gorm:"type:varchar(50);uniqueIndex" json:"sku"`
	NamaProduk      string  `gorm:"type:varchar(255);not null" json:"nama_produk"`
	Satuan          string  `gorm:"type:enum('pcs','kg','liter','kardus','ikat');not null" json:"satuan"`
	HargaBeli       float64 `gorm:"type:decimal(15,2);not null" json:"harga_beli"`
	HargaJualTunai  float64 `gorm:"type:decimal(15,2);not null" json:"harga_jual_tunai"`
	HargaJualHutang float64 `gorm:"type:decimal(15,2);not null" json:"harga_jual_hutang"`
	Stok            int     `gorm:"type:decimal(10,2);default:0" json:"stok"`

	CreatedAt time.Time      `gorm:"created_at"`
	UpdatedAt time.Time      `gorm:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type ProductRepository interface {
	AddProduct(ctx *context.Context, product *Products) error
	UpdateProduct(ctx *context.Context, product *Products, id uint) error
	DeleteProduct(ctx *context.Context, id uint) error
	GetProdcut(ctx *context.Context) (*[]Products, error)
}

type ProductUsecase interface {
	AddProductShop(ctx *context.Context, request *request.AddProduct) error
	AddBulkProductShop(ctx *context.Context, request *request.AddBulkProduct) error
	DeleteProductShop(ctx *context.Context, request *request.DeleteProduct) error
	GetAllProductShop(ctx *context.Context, request *request.GetAllProduct) (*response.GetProductResponse, error)
	UpdateProductShop(ctx *context.Context, request *request.UpdateProduct) (*response.GetAllProductResponse, error)
}

package domain

import (
	"context"
	"shop_project_be/internal/constant/enum"
	"shop_project_be/internal/dto/request"
	"shop_project_be/internal/dto/response"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Products struct {
	ID               uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SKU              string           `gorm:"type:varchar(50);uniqueIndex" json:"sku"`
	ProductName      string           `gorm:"type:varchar(255);not null" json:"product_name"`
	Unit             enum.ProductUnit `gorm:"type:smallint;check:unit IN (0,1,2,3,4);not null" json:"unit"`
	PurchasePrice    float64          `gorm:"type:decimal(15,2);not null" json:"purchase_price"`
	SellingPrice     float64          `gorm:"type:decimal(15,2);not null" json:"selling_price"`
	SellingPriceDebt float64          `gorm:"type:decimal(15,2);not null" json:"selling_price_debt"`
	Stock            int              `gorm:"type:decimal(10,2);default:0" json:"stock"`
	Category         string           `gorm:"type:varchar(100);index" json:"category"`
	Image            string           `gorm:"type:text" json:"image"`

	CreatedAt time.Time      `gorm:"created_at"`
	UpdatedAt time.Time      `gorm:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (p *Products) TableName() string {
	return "products"
}

type ProductRepository interface {
	AddProduct(ctx context.Context, product *Products) error
	UpdateProduct(ctx context.Context, product *Products, id uuid.UUID) error
	DeleteProduct(ctx context.Context, id uuid.UUID) error
	GetProdcut(ctx context.Context) (*[]Products, error)
}

type ProductUsecase interface {
	AddProductShopWithLock(ctx context.Context, request *request.AddProduct) error
	AddBulkProductShopWithLock(ctx context.Context, request *request.AddBulkProduct) error
	DeleteProductShop(ctx context.Context, request *request.DeleteProduct) error
	GetAllProductShop(ctx context.Context, request *request.GetAllProduct) (*[]response.GetProductResponse, error)
	UpdateProductShopWithLock(ctx context.Context, request *request.UpdateProduct, delta int) error
	UpdateStockWithLock(ctx context.Context, request *request.UpdateStock, delta int) error
}

package usecase

import (
	"context"
	"shop_project_be/internal/domain"
	"shop_project_be/internal/dto/request"
	"shop_project_be/internal/dto/response"

	"go.uber.org/zap"
)

type productUsecase struct {
	productRepo domain.ProductRepository
	log         *zap.Logger
}

func NewProductUsecase(productRepo domain.ProductRepository, log *zap.Logger) domain.ProductUsecase {
	return &productUsecase{
		productRepo: productRepo,
		log:         log,
	}
}

// AddBulkProductShopWithLock implements [domain.ProductUsecase].
func (p *productUsecase) AddBulkProductShopWithLock(ctx context.Context, request *request.AddBulkProduct) error {
	panic("unimplemented")
}

// AddProductShopWithLock implements [domain.ProductUsecase].
func (p *productUsecase) AddProductShopWithLock(ctx context.Context, request *request.AddProduct) error {
	panic("unimplemented")
}

// DeleteProductShop implements [domain.ProductUsecase].
func (p *productUsecase) DeleteProductShop(ctx context.Context, request *request.DeleteProduct) error {
	panic("unimplemented")
}

// GetAllProductShop implements [domain.ProductUsecase].
func (p *productUsecase) GetAllProductShop(ctx context.Context, request *request.GetAllProduct) (*[]response.GetProductResponse, error) {
	panic("unimplemented")
}

// UpdateProductShopWithLock implements [domain.ProductUsecase].
func (p *productUsecase) UpdateProductShopWithLock(ctx context.Context, request *request.UpdateProduct, delta int) error {
	panic("unimplemented")
}

// UpdateStockWithLock implements [domain.ProductUsecase].
func (p *productUsecase) UpdateStockWithLock(ctx context.Context, request *request.UpdateStock, delta int) error {
	panic("unimplemented")
}

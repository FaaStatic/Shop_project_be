package usecase

import (
	"context"
	"shop_project_be/internal/domain"
	requestdto "shop_project_be/internal/dto/request_dto"
	responsedto "shop_project_be/internal/dto/response_dto"

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

// GetProductShop implements [domain.ProductUsecase].
func (p *productUsecase) GetProductShop(ctx context.Context, request *requestdto.GetProduct) error {
	panic("unimplemented")
}

// AddBulkProductShopWithLock implements [domain.ProductUsecase].
func (p *productUsecase) AddBulkProductShopWithLock(ctx context.Context, request *requestdto.AddBulkProduct) error {
	panic("unimplemented")
}

// AddProductShopWithLock implements [domain.ProductUsecase].
func (p *productUsecase) AddProductShopWithLock(ctx context.Context, request *requestdto.AddProduct) error {
	panic("unimplemented")
}

// DeleteProductShop implements [domain.ProductUsecase].
func (p *productUsecase) DeleteProductShop(ctx context.Context, request *requestdto.DeleteProduct) error {
	panic("unimplemented")
}

// GetAllProductShop implements [domain.ProductUsecase].
func (p *productUsecase) GetAllProductShop(ctx context.Context, request *requestdto.GetAllProduct) (*[]responsedto.GetProductResponse, error) {
	panic("unimplemented")
}

// UpdateProductShopWithLock implements [domain.ProductUsecase].
func (p *productUsecase) UpdateProductShopWithLock(ctx context.Context, request *requestdto.UpdateProduct, delta int) error {
	panic("unimplemented")
}

// UpdateStockWithLock implements [domain.ProductUsecase].
func (p *productUsecase) UpdateStockWithLock(ctx context.Context, request *requestdto.UpdateStock, delta int) error {
	panic("unimplemented")
}

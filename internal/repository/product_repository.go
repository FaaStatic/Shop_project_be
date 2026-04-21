package repository

import (
	"context"
	"shop_project_be/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) domain.ProductRepository {
	return &productRepository{db: db}
}

// AddProduct implements [domain.ProductRepository].
func (p *productRepository) AddProduct(ctx context.Context, product *domain.Products) error {
	panic("unimplemented")
}

// DeleteProduct implements [domain.ProductRepository].
func (p *productRepository) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	panic("unimplemented")
}

// GetProdcut implements [domain.ProductRepository].
func (p *productRepository) GetProdcut(ctx context.Context) (*[]domain.Products, error) {
	panic("unimplemented")
}

// UpdateProduct implements [domain.ProductRepository].
func (p *productRepository) UpdateProduct(ctx context.Context, product *domain.Products, id uuid.UUID) error {
	panic("unimplemented")
}

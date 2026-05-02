package repository

import (
	"context"
	"fmt"
	"math"
	"shop_project_be/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type productRepository struct {
	db *gorm.DB
}

// GetAllProduct implements [domain.ProductRepository].
func (p *productRepository) GetAllProduct(ctx context.Context, filter domain.FilterAllProduct) (*[]domain.Products, error) {
	var itemList *[]domain.Products
	var total int64

	query := p.db.WithContext(ctx).Model(&domain.Products{})

	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}

	if filter.Search != "" {
		query = query.Where("product_name LIKE ?", "%"+filter.Search+"%")
	}

	query.Count(&total)
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit
	result := query.Offset(offset).Limit(filter.Limit).Order("product_name " + filter.Order).Find(&itemList)
	if result.Error != nil {
		return nil, result.Error
	}
	totalPages := int(math.Ceil(float64(total) / float64(filter.Limit)))
	return itemList, nil
}

// GetProduct implements [domain.ProductRepository].
func (p *productRepository) GetProduct(ctx context.Context, id uuid.UUID) (*domain.Products, error) {
	panic("unimplemented")
}

func NewProductRepository(db *gorm.DB) domain.ProductRepository {
	return &productRepository{db: db}
}

// AddProduct implements [domain.ProductRepository].
func (p *productRepository) AddProduct(ctx context.Context, product *domain.Products) error {
	result := p.db.WithContext(ctx).Create(product)
	if result.Error != nil {
		return fmt.Errorf("Failed create product: %w", result.Error)
	}
	return nil
}

// DeleteProduct implements [domain.ProductRepository].
func (p *productRepository) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	result := p.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Products{})
	if result.Error != nil {
		return fmt.Errorf("Failed delete product: %w", result.Error)
	}
	return nil
}

// UpdateProduct implements [domain.ProductRepository].
func (p *productRepository) UpdateProduct(ctx context.Context, product *domain.Products, id uuid.UUID) error {
	panic("unimplemented")
}

package repository

import (
	"context"
	"errors"
	"fmt"
	"shop_project_be/internal/constant/paginated"
	"shop_project_be/internal/domain"
	"sort"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) domain.ProductRepository {
	return &productRepository{db: db}
}

// AddBulkProduct implements [domain.ProductRepository].
func (p *productRepository) AddBulkProduct(ctx context.Context, products []*domain.Products) (*domain.BulkInsertResult, error) {
	if len(products) == 0 {
		return &domain.BulkInsertResult{}, nil
	}
	skus := make([]string, 0, len(products))
	for _, p := range products {
		skus = append(skus, p.SKU)
	}
	var existingSKUs []string
	if err := p.db.WithContext(ctx).
		Model(&domain.Products{}).
		Where("sku IN ?", skus).
		Pluck("sku", &existingSKUs).Error; err != nil {
		return nil, fmt.Errorf("failed to check existing SKUs: %w", err)
	}

	existingMap := make(map[string]struct{}, len(existingSKUs))
	for _, sku := range existingSKUs {
		existingMap[sku] = struct{}{}
	}
	newProducts := make([]*domain.Products, 0, len(products))
	skippedSKUs := make([]string, 0)
	for _, product := range products {
		if _, isDuplicate := existingMap[product.SKU]; isDuplicate {
			skippedSKUs = append(skippedSKUs, product.SKU)
		} else {
			newProducts = append(newProducts, product)
		}
	}
	if len(newProducts) == 0 {
		return &domain.BulkInsertResult{
			TotalInserted: 0,
			TotalSkipped:  len(skippedSKUs),
			SkippedSKUs:   skippedSKUs,
		}, nil
	}

	// Urutkan berdasarkan SKU agar setiap transaksi mengunci baris/index dalam
	// urutan yang sama -> mencegah deadlock saat ada upload bersamaan.
	sort.Slice(newProducts, func(i, j int) bool {
		return newProducts[i].SKU < newProducts[j].SKU
	})

	const batchSize = 100
	var inserted int64
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// ON CONFLICT (sku) DO NOTHING: aman bila ada SKU yang lolos pre-check
		// namun sudah ditambahkan transaksi lain (race) atau soft-deleted.
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "sku"}},
			DoNothing: true,
		}).CreateInBatches(newProducts, batchSize)
		if result.Error != nil {
			return fmt.Errorf("failed during batch insert: %w", result.Error)
		}
		inserted = result.RowsAffected
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Baris yang dilewati karena konflik SKU (race/soft-deleted) dihitung sebagai skipped.
	conflictSkipped := len(newProducts) - int(inserted)
	if conflictSkipped < 0 {
		conflictSkipped = 0
	}

	return &domain.BulkInsertResult{
		TotalInserted: int(inserted),
		TotalSkipped:  len(skippedSKUs) + conflictSkipped,
		SkippedSKUs:   skippedSKUs,
	}, nil
}

// AddProduct implements [domain.ProductRepository].
func (p *productRepository) AddProduct(ctx context.Context, product *domain.Products) error {
	result := p.db.WithContext(ctx).Create(product)
	if result.Error != nil {
		return fmt.Errorf("failed to add product: %w", result.Error)
	}
	return nil
}

// DeleteProduct implements [domain.ProductRepository].
func (p *productRepository) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	result := p.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Products{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete product: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("product with id %s not found", id)
	}
	return nil
}

// GetAllProduct implements [domain.ProductRepository].
func (p *productRepository) GetAllProduct(ctx context.Context, filter domain.FilterAllProduct) (*domain.PaginatedItem, error) {

	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 10
	}
	order := "DESC"
	if strings.ToUpper(filter.Order) == "DESC" {
		order = "DESC"
	}

	query := p.db.WithContext(ctx).Model(&domain.Products{})

	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}

	if filter.Search != "" {
		query = query.Where("product_name LIKE ?", "%"+filter.Search+"%")
	}

	if filter.Cursor != nil {
		if order == "ASC" {
			query = query.Where("(created_at > ?) OR (created_at = ? AND id > ?)",
				filter.Cursor.AfterTime,
				filter.Cursor.AfterTime,
				filter.Cursor.AfterID,
			)
		} else {
			query = query.Where("(created_at < ?) OR (created_at = ? AND id < ?)",
				filter.Cursor.AfterTime,
				filter.Cursor.AfterTime,
				filter.Cursor.AfterID,
			)
		}
	}
	var itemList []*domain.Products
	result := query.Order("created_at " + order + ", id " + order).Limit(filter.Limit + 1).Find(&itemList)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to get products: %w", result.Error)
	}
	hasNext := len(itemList) > filter.Limit
	if hasNext {
		itemList = itemList[:filter.Limit]
	}
	var nextCursor *paginated.CursorMeta
	if hasNext && len(itemList) > 0 {
		last := itemList[len(itemList)-1]
		nextCursor = &paginated.CursorMeta{
			AfterTime: last.CreatedAt,
			AfterID:   last.ID,
		}
	}
	return &domain.PaginatedItem{
		DataItem: itemList,
		HasNext:  hasNext,
		Cursor:   nextCursor,
	}, nil
}

// GetProduct implements [domain.ProductRepository].
func (p *productRepository) GetProduct(ctx context.Context, id uuid.UUID) (*domain.Products, error) {
	var item domain.Products
	result := p.db.WithContext(ctx).Where("id = ?", id).First(&item)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("product with id %s not found: %w", id, result.Error)
		}
		return nil, fmt.Errorf("failed to get product: %w", result.Error)
	}
	return &item, nil
}

// UpdateProduct implements [domain.ProductRepository].
func (p *productRepository) UpdateProduct(ctx context.Context, product *domain.Products, id uuid.UUID) error {
	result := p.db.WithContext(ctx).
		Model(&domain.Products{}).
		Where("id = ?", id).
		Updates(product)
	if result.Error != nil {
		return fmt.Errorf("failed to update product: %w", result.Error)
	}
	return nil
}

// UpdateProductWithLock implements [domain.ProductRepository].
// Mengunci baris produk (SELECT ... FOR UPDATE) lalu menerapkan update parsial
// dari fields. Bila stockDelta != 0, stok diubah secara atomik (current + delta)
// di dalam lock yang sama -> mencegah lost-update saat ada perubahan bersamaan.
func (p *productRepository) UpdateProductWithLock(ctx context.Context, id uuid.UUID, fields map[string]interface{}, stockDelta float64) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var product domain.Products
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", id).First(&product).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("product with id %s not found", id)
			}
			return fmt.Errorf("failed to find product for update: %w", err)
		}

		if stockDelta != 0 {
			newStock := product.Stock + stockDelta
			if newStock < 0 {
				return fmt.Errorf("insufficient stock for product %s (current: %v, requested change: %v)", id, product.Stock, stockDelta)
			}
			fields["stock"] = newStock
		}

		if len(fields) == 0 {
			return nil
		}

		if err := tx.Model(&domain.Products{}).Where("id = ?", id).Updates(fields).Error; err != nil {
			return fmt.Errorf("failed to update product: %w", err)
		}
		return nil
	})
}

// UpdateStockWithLock implements [domain.ProductRepository].
func (p *productRepository) UpdateStockWithLock(ctx context.Context, id uuid.UUID, delta float64) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var product domain.Products
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&product)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return fmt.Errorf("product with id %s not found", id)
			}
			return fmt.Errorf("failed to find product for update: %w", result.Error)
		}

		newStock := product.Stock + delta

		if newStock < 0 {
			return fmt.Errorf("insufficient stock for product %s (current: %v, requested change: %v)", id, product.Stock, delta)
		}

		updateResult := tx.Model(&domain.Products{}).
			Where("id = ?", id).
			Update("stock", newStock)

		if updateResult.Error != nil {
			return fmt.Errorf("failed to update product stock: %w", updateResult.Error)
		}

		return nil
	})
}

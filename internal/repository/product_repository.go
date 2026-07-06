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

	// Sort by SKU so every transaction locks rows/indexes in
	// the same order -> preventing deadlock during concurrent uploads.
	sort.Slice(newProducts, func(i, j int) bool {
		return newProducts[i].SKU < newProducts[j].SKU
	})

	const batchSize = 100
	var inserted int64
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// ON CONFLICT (sku) DO NOTHING: safe if a SKU passed the pre-check
		// but was already added by another transaction (race) or soft-deleted.
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

	// Rows skipped due to a SKU conflict (race/soft-deleted) are counted as skipped.
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
	if strings.ToUpper(filter.Order) == "ASC" {
		order = "ASC"
	}

	query := p.db.WithContext(ctx).Model(&domain.Products{})

	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}

	if filter.Search != "" {
		escaped := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(filter.Search)
		query = query.Where("product_name LIKE ? ESCAPE '\\'", "%"+escaped+"%")
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
	if result.RowsAffected == 0 {
		return fmt.Errorf("product with id %s not found", id)
	}
	return nil
}

// UpdateProductWithLock implements [domain.ProductRepository].
// Locks the product row (SELECT ... FOR UPDATE) then applies a partial update
// from fields. If stockDelta != 0, stock changes atomically (current + delta)
// within the same lock -> preventing lost updates under concurrent changes.
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

// ReserveStock implements [domain.ProductRepository]. All items are deducted
// in one DB transaction with a per-product row lock: all succeed or
// none at all (insufficient stock -> the whole reservation is aborted).
func (p *productRepository) ReserveStock(ctx context.Context, items []domain.PaymentItem) error {
	if len(items) == 0 {
		return nil
	}
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock all product rows up-front in a deterministic id order
		// (ORDER BY id) so two concurrent reservations sharing products do not
		// lock in crossing order -> preventing deadlock. The loop below stays in
		// the original item order so validation & error messages are unchanged.
		if err := lockProductsOrdered(tx, paymentItemIDs(items)); err != nil {
			return err
		}
		for _, it := range items {
			var product domain.Products
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("id = ?", it.ProductID).First(&product).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("product with id %s not found", it.ProductID)
				}
				return fmt.Errorf("failed to lock product: %w", err)
			}
			if product.Stock < it.Qty {
				return fmt.Errorf("insufficient stock for product %s (current: %v, requested: %v)", it.ProductID, product.Stock, it.Qty)
			}
			if err := tx.Model(&domain.Products{}).Where("id = ?", it.ProductID).
				Update("stock", product.Stock-it.Qty).Error; err != nil {
				return fmt.Errorf("failed to reserve stock: %w", err)
			}
		}
		return nil
	})
}

// RestoreStock implements [domain.ProductRepository]. Returns stock
// by the reserved amount without an upper-bound check (the exact inverse of ReserveStock).
func (p *productRepository) RestoreStock(ctx context.Context, items []domain.PaymentItem) error {
	if len(items) == 0 {
		return nil
	}
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock rows in a deterministic id order first so concurrent restores
		// sharing products do not deadlock (missing rows are ignored,
		// exactly as with the earlier lock-free UPDATE).
		if err := lockProductsOrdered(tx, paymentItemIDs(items)); err != nil {
			return err
		}
		for _, it := range items {
			if err := tx.Model(&domain.Products{}).Where("id = ?", it.ProductID).
				Update("stock", gorm.Expr("stock + ?", it.Qty)).Error; err != nil {
				return fmt.Errorf("failed to restore stock: %w", err)
			}
		}
		return nil
	})
}

// paymentItemIDs collects the product ids from a list of payment items.
func paymentItemIDs(items []domain.PaymentItem) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ProductID)
	}
	return ids
}

// lockProductsOrdered locks (SELECT ... FOR UPDATE) the product rows for ids
// in a deterministic id order (ORDER BY id). Because all concurrent
// transactions acquire locks in the same order, a wait-for cycle between
// rows cannot form -> preventing deadlock. Rows that do not exist
// are simply ignored; the caller handles "not found" via the per-item query.
func lockProductsOrdered(tx *gorm.DB, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	var locked []domain.Products
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id IN ?", ids).Order("id").Find(&locked).Error; err != nil {
		return fmt.Errorf("failed to lock products: %w", err)
	}
	return nil
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

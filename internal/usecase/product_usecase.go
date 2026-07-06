package usecase

import (
	"context"
	"fmt"
	"shop_project_be/internal/constant/enum"
	"shop_project_be/internal/constant/paginated"
	"shop_project_be/internal/domain"
	requestdto "shop_project_be/internal/dto/request_dto"
	responsedto "shop_project_be/internal/dto/response_dto"
	"shop_project_be/pkg/sheet"
	"strings"
	"time"

	"github.com/google/uuid"
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
func (p *productUsecase) GetProductShop(ctx context.Context, request *requestdto.GetProduct) (*domain.Products, error) {
	productUid, errUid := uuid.Parse(request.ID)
	if errUid != nil {
		p.log.Error("failed to parse product id", zap.Error(errUid))
		return nil, fmt.Errorf("invalid product id format")
	}
	products, err := p.productRepo.GetProduct(ctx, productUid)
	if err != nil {
		p.log.Error("failed to get product", zap.Error(err))
		return nil, fmt.Errorf("failed to get product")
	}
	return products, nil
}

// AddBulkProductShopWithLock implements [domain.ProductUsecase].
// Adds many products from an uploaded CSV/Excel file. Steps:
//  1. parse the file into product rows,
//  2. dedupe duplicate SKUs within the file (keep the first occurrence),
//  3. validate each row's unit,
//  4. hand off to the repo, which checks existing SKUs + batch-inserts
//     in sorted order with ON CONFLICT DO NOTHING (anti-deadlock & anti-duplicate).
func (p *productUsecase) AddBulkProductShopWithLock(ctx context.Context, request *requestdto.AddBulkProduct) error {
	if request.FileUpload == nil {
		return fmt.Errorf("file upload is required")
	}

	file, err := request.FileUpload.Open()
	if err != nil {
		p.log.Error("failed to open uploaded file", zap.Error(err))
		return fmt.Errorf("failed to open uploaded file")
	}
	defer file.Close()

	rows, rowErrors, err := sheet.ParseProducts(file, request.FileUpload.Filename)
	if err != nil {
		p.log.Error("failed to parse uploaded file", zap.Error(err))
		return fmt.Errorf("failed to parse file: %w", err)
	}

	seen := make(map[string]struct{}, len(rows))
	products := make([]*domain.Products, 0, len(rows))
	var duplicateInFile []string

	for _, row := range rows {
		if _, exists := seen[row.SKU]; exists {
			duplicateInFile = append(duplicateInFile, row.SKU)
			continue
		}
		seen[row.SKU] = struct{}{}

		unit, err := enum.ParseProductUnit(row.Unit)
		if err != nil {
			rowErrors = append(rowErrors, sheet.RowError{Line: row.Line, Message: "invalid unit"})
			continue
		}
		products = append(products, &domain.Products{
			SKU:              row.SKU,
			ProductName:      row.ProductName,
			Unit:             unit,
			PurchasePrice:    row.PurchasePrice,
			SellingPrice:     row.SellingPrice,
			SellingPriceDebt: row.SellingPriceDebt,
			Stock:            row.Stock,
			Category:         row.Category,
			Image:            row.Image,
		})
	}

	if len(products) == 0 {
		p.log.Warn("no valid product to insert",
			zap.Int("row_errors", len(rowErrors)),
			zap.Int("duplicate_in_file", len(duplicateInFile)),
		)
		return fmt.Errorf("no valid product to import")
	}

	result, err := p.productRepo.AddBulkProduct(ctx, products)
	if err != nil {
		p.log.Error("failed to bulk insert products", zap.Error(err))
		return fmt.Errorf("failed to import products")
	}

	p.log.Info("bulk product import finished",
		zap.Int("inserted", result.TotalInserted),
		zap.Int("skipped_existing", result.TotalSkipped),
		zap.Int("duplicate_in_file", len(duplicateInFile)),
		zap.Int("row_errors", len(rowErrors)),
	)

	return nil
}

// AddProductShopWithLock implements [domain.ProductUsecase].
func (p *productUsecase) AddProductShopWithLock(ctx context.Context, request *requestdto.AddProduct) error {
	err := p.productRepo.AddProduct(ctx, &domain.Products{
		SKU:              request.SKU,
		ProductName:      request.ProductName,
		Unit:             enum.ProductUnit(request.Unit),
		PurchasePrice:    request.PurchasePrice,
		SellingPrice:     request.SellingPrice,
		SellingPriceDebt: request.SellingPriceDebt,
		Stock:            request.Stock,
		Category:         request.Category,
		Image:            request.Image,
	})
	if err != nil {
		p.log.Error("failed to add product", zap.Error(err))
		return fmt.Errorf("failed to add product")
	}
	return nil
}

// DeleteProductShop implements [domain.ProductUsecase].
func (p *productUsecase) DeleteProductShop(ctx context.Context, request *requestdto.DeleteProduct) error {
	id, errId := uuid.Parse(request.ID)
	if errId != nil {
		p.log.Error("failed to delete product", zap.Error(errId))
		return fmt.Errorf("failed to delete product")
	}
	err := p.productRepo.DeleteProduct(ctx, id)
	if err != nil {
		p.log.Error("failed to delete product", zap.Error(err))
		return fmt.Errorf("failed to delete product")
	}
	return nil
}

// GetAllProductShop implements [domain.ProductUsecase].
// Fetches the product list with search support, category filter, and
// cursor pagination (last_id + after_time from the previous page's result).
func (p *productUsecase) GetAllProductShop(ctx context.Context, request *requestdto.GetAllProduct) (*responsedto.GetAllProductResponse, error) {
	// Cursor pagination is optional. On the first page the frontend does not yet have
	// last_id/after_time, so empty/absent parameters are treated as
	// "no cursor" (not an error). The cursor is used only when both are set.
	var lastId, afterTimeRaw string
	if request.LastId != nil {
		lastId = strings.TrimSpace(*request.LastId)
	}
	if request.AfterTime != nil {
		afterTimeRaw = strings.TrimSpace(*request.AfterTime)
	}

	var cursor *paginated.CursorMeta
	if lastId != "" && afterTimeRaw != "" {
		afterTime, err := time.Parse(paginated.TimeLayout, afterTimeRaw)
		if err != nil {
			p.log.Error("failed to parse after_time", zap.Error(err))
			return nil, fmt.Errorf("invalid after_time format")
		}
		lastID, err := uuid.Parse(lastId)
		if err != nil {
			p.log.Error("failed to parse last_id", zap.Error(err))
			return nil, fmt.Errorf("invalid last_id format")
		}
		cursor = &paginated.CursorMeta{
			AfterTime: afterTime,
			AfterID:   lastID,
		}
	}

	filter := domain.FilterAllProduct{
		Search:   request.Search,
		Category: request.Category,
		Cursor:   cursor,
		Limit:    request.Limit,
		Order:    request.Order,
	}

	result, err := p.productRepo.GetAllProduct(ctx, filter)
	if err != nil {
		p.log.Error("failed to get all products", zap.Error(err))
		return nil, fmt.Errorf("failed to get all products")
	}

	products := make([]responsedto.ProductDtoResponse, 0, len(result.DataItem))
	for _, item := range result.DataItem {
		products = append(products, responsedto.ProductDtoResponse{
			ID:               item.ID,
			SKU:              item.SKU,
			ProductName:      item.ProductName,
			Unit:             int(item.Unit),
			PurchasePrice:    item.PurchasePrice,
			SellingPrice:     item.SellingPrice,
			SellingPriceDebt: item.SellingPriceDebt,
			Stock:            item.Stock,
			Category:         item.Category,
			Image:            item.Image,
			UpdatedAt:        item.UpdatedAt,
		})
	}

	// The cursor is nil on the last page (HasNext=false). Encode is nil-safe
	// and returns empty strings instead of a nil-pointer panic.
	nextId, nextTime := result.Cursor.Encode()

	responses := responsedto.GetAllProductResponse{
		UserId:      request.UserId,
		NextId:      nextId,
		NextTime:    nextTime,
		HasNext:     result.HasNext,
		ProductList: products,
	}
	return &responses, nil
}

// UpdateProductShopWithLock implements [domain.ProductUsecase].
// Updates product attributes under a row-level lock so concurrent changes
// don't overwrite each other. Stock changes go via delta (atomic within the
// lock); the DTO's stock field is not used here to avoid two sources of truth.
func (p *productUsecase) UpdateProductShopWithLock(ctx context.Context, request *requestdto.UpdateProduct, delta float64) error {
	id, err := uuid.Parse(request.ID)
	if err != nil {
		p.log.Error("failed to parse product id", zap.Error(err))
		return fmt.Errorf("invalid product id format")
	}

	fields := make(map[string]interface{})
	if request.SKU != nil {
		fields["sku"] = *request.SKU
	}
	if request.ProductName != nil {
		fields["product_name"] = *request.ProductName
	}
	if request.Unit != nil {
		fields["unit"] = enum.ProductUnit(*request.Unit)
	}
	if request.PurchasePrice != nil {
		fields["purchase_price"] = *request.PurchasePrice
	}
	if request.SellingPrice != nil {
		fields["selling_price"] = *request.SellingPrice
	}
	if request.SellingPriceDebt != nil {
		fields["selling_price_debt"] = *request.SellingPriceDebt
	}
	if request.Category != nil {
		fields["category"] = *request.Category
	}
	if request.Image != nil {
		fields["image"] = *request.Image
	}

	if len(fields) == 0 && delta == 0 {
		return fmt.Errorf("no fields to update")
	}

	if err := p.productRepo.UpdateProductWithLock(ctx, id, fields, delta); err != nil {
		p.log.Error("failed to update product", zap.Error(err))
		return fmt.Errorf("failed to update product: %w", err)
	}
	return nil
}

// UpdateStockWithLock implements [domain.ProductUsecase].
// Increments/decrements product stock by delta atomically with a row-level
// lock (SELECT ... FOR UPDATE) in the repository, so it's safe from race/deadlock.
func (p *productUsecase) UpdateStockWithLock(ctx context.Context, request *requestdto.UpdateStock, delta float64) error {
	id, err := uuid.Parse(request.ID)
	if err != nil {
		p.log.Error("failed to parse product id", zap.Error(err))
		return fmt.Errorf("invalid product id format")
	}

	if err := p.productRepo.UpdateStockWithLock(ctx, id, delta); err != nil {
		p.log.Error("failed to update stock", zap.Error(err))
		return fmt.Errorf("failed to update stock: %w", err)
	}
	return nil
}

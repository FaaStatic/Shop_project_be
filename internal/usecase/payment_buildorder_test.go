package usecase

import (
	"context"
	"strings"
	"testing"

	"shop_project_be/internal/constant/enum"
	"shop_project_be/internal/domain"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// fakeProductRepo embeds domain.ProductRepository so only the methods used by
// buildOrder (GetProduct) need to be implemented for these tests.
type fakeProductRepo struct {
	domain.ProductRepository
	product *domain.Products
}

func (f fakeProductRepo) GetProduct(ctx context.Context, id uuid.UUID) (*domain.Products, error) {
	p := *f.product
	p.ID = id
	return &p, nil
}

func TestBuildOrder_RejectsDigitalProduct(t *testing.T) {
	u := &paymentUsecase{
		productRepo: fakeProductRepo{product: &domain.Products{
			ProductType:  enum.Digital,
			SellingPrice: 10000,
			Stock:        0,
		}},
		log: zap.NewNop(),
	}

	_, _, err := u.buildOrder(context.Background(), []itemPair{{productID: uuid.NewString(), qty: 1}})
	if err == nil {
		t.Fatal("expected buildOrder to reject a digital product, got nil error")
	}
	if !strings.Contains(err.Error(), "digital") {
		t.Fatalf("expected error to mention digital product, got: %v", err)
	}
}

func TestBuildOrder_AllowsPhysicalProduct(t *testing.T) {
	u := &paymentUsecase{
		productRepo: fakeProductRepo{product: &domain.Products{
			ProductType:  enum.Physical,
			SellingPrice: 10000,
			Stock:        5,
		}},
		log: zap.NewNop(),
	}

	gross, items, err := u.buildOrder(context.Background(), []itemPair{{productID: uuid.NewString(), qty: 1}})
	if err != nil {
		t.Fatalf("expected no error for physical product, got: %v", err)
	}
	if gross != 10000 {
		t.Fatalf("expected gross 10000, got %d", gross)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 payment item, got %d", len(items))
	}
}

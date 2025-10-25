package repo

import (
	"context"
	"go-cron/models"
)

// ProductRepositoryInterface defines the interface for product repository operations
type ProductRepositoryInterface interface {
	GetAllProducts(ctx context.Context) ([]models.Product, error)
	GetProductByTitle(ctx context.Context, title string) (*models.Product, error)
	CreateProduct(ctx context.Context, title, handle string) (int, error)
	UpdateProduct(ctx context.Context, id int, title, handle string) error
	CreateProductsBatch(ctx context.Context, products []struct{ Title, Handle string }) error
	UpdateProductsBatch(ctx context.Context, updates []struct {
		ID     int
		Title  string
		Handle string
	}) error
}

// Ensure ProductRepository implements the interface
var _ ProductRepositoryInterface = (*ProductRepository)(nil)

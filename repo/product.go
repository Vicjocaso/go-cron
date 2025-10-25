package repo

import (
	"context"
	"database/sql"
	"fmt"
	"go-cron/models"
)

// ProductRepository handles database operations for products
type ProductRepository struct {
	db *sql.DB
}

// NewProductRepository creates a new product repository
func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// GetAllProducts fetches all products from the database
func (r *ProductRepository) GetAllProducts(ctx context.Context) ([]models.Product, error) {
	query := `SELECT id, title, COALESCE(handle, '') as handle FROM products ORDER BY id`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Title, &p.Handle); err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating products: %w", err)
	}

	return products, nil
}

// GetProductByTitle finds a product by its title (case-insensitive)
func (r *ProductRepository) GetProductByTitle(ctx context.Context, title string) (*models.Product, error) {
	query := `SELECT id, title, COALESCE(handle, '') as handle FROM products WHERE LOWER(title) = LOWER($1)`

	var p models.Product
	err := r.db.QueryRowContext(ctx, query, title).Scan(&p.ID, &p.Title, &p.Handle)
	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query product by title: %w", err)
	}

	return &p, nil
}

// CreateProduct inserts a new product into the database
// If a duplicate handle exists, it will be skipped gracefully
func (r *ProductRepository) CreateProduct(ctx context.Context, title, handle string) (int, error) {
	query := `
		INSERT INTO products (title, handle) 
		VALUES ($1, $2) 
		ON CONFLICT (handle) DO NOTHING
		RETURNING id`

	var newID int
	err := r.db.QueryRowContext(ctx, query, title, handle).Scan(&newID)
	if err == sql.ErrNoRows {
		// Duplicate was skipped, return 0 to indicate no insertion
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to create product: %w", err)
	}

	return newID, nil
}

// UpdateProduct updates an existing product
func (r *ProductRepository) UpdateProduct(ctx context.Context, id int, title, handle string) error {
	query := `UPDATE products SET title = $1, handle = $2 WHERE id = $3`

	result, err := r.db.ExecContext(ctx, query, title, handle, id)
	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no product found with id %d", id)
	}

	return nil
}

// CreateProductsBatch creates multiple products in a single transaction for better performance
// Duplicates (based on handle) are automatically skipped without errors
func (r *ProductRepository) CreateProductsBatch(ctx context.Context, products []struct{ Title, Handle string }) error {
	if len(products) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Use ON CONFLICT to skip duplicates gracefully
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO products (title, handle) 
		VALUES ($1, $2) 
		ON CONFLICT (handle) DO NOTHING`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, p := range products {
		if _, err := stmt.ExecContext(ctx, p.Title, p.Handle); err != nil {
			return fmt.Errorf("failed to insert product %s: %w", p.Title, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateProductsBatch updates multiple products in a single transaction
func (r *ProductRepository) UpdateProductsBatch(ctx context.Context, updates []struct {
	ID     int
	Title  string
	Handle string
}) error {
	if len(updates) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `UPDATE products SET title = $1, handle = $2 WHERE id = $3`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, u := range updates {
		if _, err := stmt.ExecContext(ctx, u.Title, u.Handle, u.ID); err != nil {
			return fmt.Errorf("failed to update product %d: %w", u.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

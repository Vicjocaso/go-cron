package repo

import (
	"context"
	"go-cron/models"
	"testing"
)

// MockProductRepository is a mock implementation of ProductRepositoryInterface for testing
type MockProductRepository struct {
	GetAllProductsFunc      func(ctx context.Context) ([]models.Product, error)
	GetProductByTitleFunc   func(ctx context.Context, title string) (*models.Product, error)
	CreateProductFunc       func(ctx context.Context, title, handle string) (int, error)
	UpdateProductFunc       func(ctx context.Context, id int, title, handle string) error
	CreateProductsBatchFunc func(ctx context.Context, products []struct{ Title, Handle string }) error
	UpdateProductsBatchFunc func(ctx context.Context, updates []struct {
		ID     int
		Title  string
		Handle string
	}) error
}

func (m *MockProductRepository) GetAllProducts(ctx context.Context) ([]models.Product, error) {
	if m.GetAllProductsFunc != nil {
		return m.GetAllProductsFunc(ctx)
	}
	return []models.Product{}, nil
}

func (m *MockProductRepository) GetProductByTitle(ctx context.Context, title string) (*models.Product, error) {
	if m.GetProductByTitleFunc != nil {
		return m.GetProductByTitleFunc(ctx, title)
	}
	return nil, nil
}

func (m *MockProductRepository) CreateProduct(ctx context.Context, title, handle string) (int, error) {
	if m.CreateProductFunc != nil {
		return m.CreateProductFunc(ctx, title, handle)
	}
	return 0, nil
}

func (m *MockProductRepository) UpdateProduct(ctx context.Context, id int, title, handle string) error {
	if m.UpdateProductFunc != nil {
		return m.UpdateProductFunc(ctx, id, title, handle)
	}
	return nil
}

func (m *MockProductRepository) CreateProductsBatch(ctx context.Context, products []struct{ Title, Handle string }) error {
	if m.CreateProductsBatchFunc != nil {
		return m.CreateProductsBatchFunc(ctx, products)
	}
	return nil
}

func (m *MockProductRepository) UpdateProductsBatch(ctx context.Context, updates []struct {
	ID     int
	Title  string
	Handle string
}) error {
	if m.UpdateProductsBatchFunc != nil {
		return m.UpdateProductsBatchFunc(ctx, updates)
	}
	return nil
}

// Test_SyncService_CompareAndSync_NewItems tests creating new items
func Test_SyncService_CompareAndSync_NewItems(t *testing.T) {
	ctx := context.Background()

	// Mock repository with empty database
	mockRepo := &MockProductRepository{
		GetAllProductsFunc: func(ctx context.Context) ([]models.Product, error) {
			return []models.Product{}, nil // Empty database
		},
		CreateProductsBatchFunc: func(ctx context.Context, products []struct{ Title, Handle string }) error {
			// Verify we're creating the right products
			if len(products) != 3 {
				t.Errorf("Expected 3 products to create, got %d", len(products))
			}
			expectedTitles := map[string]bool{
				"Product A": true,
				"Product B": true,
				"Product C": true,
			}
			for _, p := range products {
				if !expectedTitles[p.Title] {
					t.Errorf("Unexpected product title: %s", p.Title)
				}
			}
			return nil
		},
	}

	syncService := NewSyncService(mockRepo)

	// External API data (mock)
	externalItems := []map[string]interface{}{
		{"ItemName": "Product A", "ItemCode": "A001"},
		{"ItemName": "Product B", "ItemCode": "B001"},
		{"ItemName": "Product C", "ItemCode": "C001"},
	}

	result, err := syncService.CompareAndSync(ctx, externalItems)
	if err != nil {
		t.Fatalf("CompareAndSync failed: %v", err)
	}

	// Assertions
	if result.Created != 3 {
		t.Errorf("Expected 3 items created, got %d", result.Created)
	}
	if result.Updated != 0 {
		t.Errorf("Expected 0 items updated, got %d", result.Updated)
	}
	if result.Unchanged != 0 {
		t.Errorf("Expected 0 items unchanged, got %d", result.Unchanged)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}
}

// Test_SyncService_CompareAndSync_UpdateExisting tests updating existing items
func Test_SyncService_CompareAndSync_UpdateExisting(t *testing.T) {
	ctx := context.Background()

	// Mock repository with existing products (with old handles)
	mockRepo := &MockProductRepository{
		GetAllProductsFunc: func(ctx context.Context) ([]models.Product, error) {
			return []models.Product{
				{ID: 1, Title: "Product A", Handle: "old-handle-a"},
				{ID: 2, Title: "Product B", Handle: "old-handle-b"},
			}, nil
		},
		UpdateProductsBatchFunc: func(ctx context.Context, updates []struct {
			ID     int
			Title  string
			Handle string
		}) error {
			// Verify we're updating the right products
			if len(updates) != 2 {
				t.Errorf("Expected 2 products to update, got %d", len(updates))
			}
			for _, u := range updates {
				if u.Title == "Product A" && u.Handle != "product-a" {
					t.Errorf("Expected handle 'product-a', got '%s'", u.Handle)
				}
				if u.Title == "Product B" && u.Handle != "product-b" {
					t.Errorf("Expected handle 'product-b', got '%s'", u.Handle)
				}
			}
			return nil
		},
	}

	syncService := NewSyncService(mockRepo)

	// External API data with same titles but updated data
	externalItems := []map[string]interface{}{
		{"ItemName": "Product A", "ItemCode": "A001"},
		{"ItemName": "Product B", "ItemCode": "B001"},
	}

	result, err := syncService.CompareAndSync(ctx, externalItems)
	if err != nil {
		t.Fatalf("CompareAndSync failed: %v", err)
	}

	// Assertions
	if result.Created != 0 {
		t.Errorf("Expected 0 items created, got %d", result.Created)
	}
	if result.Updated != 2 {
		t.Errorf("Expected 2 items updated, got %d", result.Updated)
	}
	if result.Unchanged != 0 {
		t.Errorf("Expected 0 items unchanged, got %d", result.Unchanged)
	}
}

// Test_SyncService_CompareAndSync_UnchangedItems tests items that don't need updates
func Test_SyncService_CompareAndSync_UnchangedItems(t *testing.T) {
	ctx := context.Background()

	// Mock repository with products matching external API
	mockRepo := &MockProductRepository{
		GetAllProductsFunc: func(ctx context.Context) ([]models.Product, error) {
			return []models.Product{
				{ID: 1, Title: "Product A", Handle: "product-a"},
				{ID: 2, Title: "Product B", Handle: "product-b"},
			}, nil
		},
	}

	syncService := NewSyncService(mockRepo)

	// External API data matching database
	externalItems := []map[string]interface{}{
		{"ItemName": "Product A", "ItemCode": "A001"},
		{"ItemName": "Product B", "ItemCode": "B001"},
	}

	result, err := syncService.CompareAndSync(ctx, externalItems)
	if err != nil {
		t.Fatalf("CompareAndSync failed: %v", err)
	}

	// Assertions
	if result.Created != 0 {
		t.Errorf("Expected 0 items created, got %d", result.Created)
	}
	if result.Updated != 0 {
		t.Errorf("Expected 0 items updated, got %d", result.Updated)
	}
	if result.Unchanged != 2 {
		t.Errorf("Expected 2 items unchanged, got %d", result.Unchanged)
	}
}

// Test_SyncService_CompareAndSync_MixedScenario tests a realistic mixed scenario
func Test_SyncService_CompareAndSync_MixedScenario(t *testing.T) {
	ctx := context.Background()

	// Mock repository with mixed data
	mockRepo := &MockProductRepository{
		GetAllProductsFunc: func(ctx context.Context) ([]models.Product, error) {
			return []models.Product{
				{ID: 1, Title: "Existing Product 1", Handle: "existing-product-1"},
				{ID: 2, Title: "Product To Update", Handle: "old-handle"}, // Will be updated
			}, nil
		},
		CreateProductsBatchFunc: func(ctx context.Context, products []struct{ Title, Handle string }) error {
			if len(products) != 2 {
				t.Errorf("Expected 2 new products, got %d", len(products))
			}
			return nil
		},
		UpdateProductsBatchFunc: func(ctx context.Context, updates []struct {
			ID     int
			Title  string
			Handle string
		}) error {
			if len(updates) != 1 {
				t.Errorf("Expected 1 product update, got %d", len(updates))
			}
			// The title matches but handle is different (will update handle to "product-to-update")
			if updates[0].ID != 2 {
				t.Errorf("Expected ID 2, got %d", updates[0].ID)
			}
			return nil
		},
	}

	syncService := NewSyncService(mockRepo)

	// External API data: 1 unchanged, 1 updated (same title but handle changes), 2 new
	externalItems := []map[string]interface{}{
		{"ItemName": "Existing Product 1", "ItemCode": "E001"},  // Unchanged
		{"ItemName": "Product To Update", "ItemCode": "U001"},   // Updated (handle will change from "old-handle" to "product-to-update")
		{"ItemName": "Brand New Product A", "ItemCode": "N001"}, // New
		{"ItemName": "Brand New Product B", "ItemCode": "N002"}, // New
	}

	result, err := syncService.CompareAndSync(ctx, externalItems)
	if err != nil {
		t.Fatalf("CompareAndSync failed: %v", err)
	}

	// Assertions
	if result.Created != 2 {
		t.Errorf("Expected 2 items created, got %d", result.Created)
	}
	if result.Updated != 1 {
		t.Errorf("Expected 1 item updated, got %d", result.Updated)
	}
	if result.Unchanged != 1 {
		t.Errorf("Expected 1 item unchanged, got %d", result.Unchanged)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %v", result.Errors)
	}
}

// Test_SyncService_CompareAndSync_InvalidData tests handling of invalid external data
func Test_SyncService_CompareAndSync_InvalidData(t *testing.T) {
	ctx := context.Background()

	mockRepo := &MockProductRepository{
		GetAllProductsFunc: func(ctx context.Context) ([]models.Product, error) {
			return []models.Product{}, nil
		},
		CreateProductsBatchFunc: func(ctx context.Context, products []struct{ Title, Handle string }) error {
			// Should only get valid items
			if len(products) != 1 {
				t.Errorf("Expected 1 valid product, got %d", len(products))
			}
			return nil
		},
	}

	syncService := NewSyncService(mockRepo)

	// External API data with invalid items
	externalItems := []map[string]interface{}{
		{"ItemName": "", "ItemCode": "A001"},              // Invalid: empty name
		{"ItemCode": "B001"},                              // Invalid: missing ItemName
		{"ItemName": 123, "ItemCode": "C001"},             // Invalid: wrong type
		{"ItemName": "Valid Product", "ItemCode": "D001"}, // Valid
	}

	result, err := syncService.CompareAndSync(ctx, externalItems)
	if err != nil {
		t.Fatalf("CompareAndSync failed: %v", err)
	}

	// Should have 3 errors and 1 created item
	if result.Created != 1 {
		t.Errorf("Expected 1 item created, got %d", result.Created)
	}
	if len(result.Errors) != 3 {
		t.Errorf("Expected 3 errors, got %d: %v", len(result.Errors), result.Errors)
	}
}

// Test_SyncService_CompareAndSync_CaseInsensitiveMatching tests case-insensitive title matching
func Test_SyncService_CompareAndSync_CaseInsensitiveMatching(t *testing.T) {
	ctx := context.Background()

	mockRepo := &MockProductRepository{
		GetAllProductsFunc: func(ctx context.Context) ([]models.Product, error) {
			return []models.Product{
				{ID: 1, Title: "Coffee Beans", Handle: "coffee-beans"},
				{ID: 2, Title: "TEA LEAVES", Handle: "tea-leaves"},
			}, nil
		},
	}

	syncService := NewSyncService(mockRepo)

	// External API data with different casing
	externalItems := []map[string]interface{}{
		{"ItemName": "COFFEE BEANS", "ItemCode": "C001"}, // Same as "Coffee Beans"
		{"ItemName": "tea leaves", "ItemCode": "T001"},   // Same as "TEA LEAVES"
	}

	result, err := syncService.CompareAndSync(ctx, externalItems)
	if err != nil {
		t.Fatalf("CompareAndSync failed: %v", err)
	}

	// Both should be treated as unchanged (case-insensitive)
	// But they will be updated because the title casing changed
	if result.Updated != 2 {
		t.Errorf("Expected 2 items updated (case changed), got %d", result.Updated)
	}
	if result.Created != 0 {
		t.Errorf("Expected 0 items created, got %d", result.Created)
	}
}

// Test_generateHandle tests the handle generation function
func Test_generateHandle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple Product", "simple-product"},
		{"Product With CAPS", "product-with-caps"},
		{"Product_With_Underscores", "product-with-underscores"},
		{"Product   Multiple   Spaces", "product---multiple---spaces"},
		{"Product@#$%Special*&Chars", "productspecialchars"},
		{"123 Numeric Product 456", "123-numeric-product-456"},
		{"Café Latté", "caf-latt"}, // Special characters removed
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := generateHandle(tt.input)
			if result != tt.expected {
				t.Errorf("generateHandle(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Test_normalizeTitle tests the title normalization function
func Test_normalizeTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple Title", "simple title"},
		{"  Title With Spaces  ", "title with spaces"},
		{"UPPERCASE TITLE", "uppercase title"},
		{"MixedCase Title", "mixedcase title"},
		{"   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeTitle(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeTitle(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

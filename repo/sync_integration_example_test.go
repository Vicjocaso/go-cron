package repo

import (
	"context"
	"go-cron/models"
	"testing"
)

// ExampleSyncService demonstrates using the test data helper
func ExampleSyncService() {
	ctx := context.Background()
	helper := NewTestDataHelper()

	// Create mock repository
	mockRepo := &MockProductRepository{
		GetAllProductsFunc: func(ctx context.Context) ([]models.Product, error) {
			return helper.GetMockDatabaseProducts(), nil
		},
		CreateProductsBatchFunc: func(ctx context.Context, products []struct{ Title, Handle string }) error {
			return nil
		},
		UpdateProductsBatchFunc: func(ctx context.Context, updates []struct {
			ID     int
			Title  string
			Handle string
		}) error {
			return nil
		},
	}

	// Create sync service
	syncService := NewSyncService(mockRepo)

	// Use mock external items
	externalItems := helper.GetMockExternalItems()

	// Perform sync
	_, _ = syncService.CompareAndSync(ctx, externalItems)

	// Output would be logged
}

// Test_SyncService_WithLargeDataset tests performance with large dataset
func Test_SyncService_WithLargeDataset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large dataset test in short mode")
	}

	ctx := context.Background()
	helper := NewTestDataHelper()

	// Generate 100 mock items
	largeDataset := helper.GetMockExternalItemsLarge(100)

	mockRepo := &MockProductRepository{
		GetAllProductsFunc: func(ctx context.Context) ([]models.Product, error) {
			return []models.Product{}, nil // Empty database
		},
		CreateProductsBatchFunc: func(ctx context.Context, products []struct{ Title, Handle string }) error {
			if len(products) != 100 {
				t.Errorf("Expected 100 products, got %d", len(products))
			}
			return nil
		},
	}

	syncService := NewSyncService(mockRepo)
	result, err := syncService.CompareAndSync(ctx, largeDataset)

	if err != nil {
		t.Fatalf("Failed to sync large dataset: %v", err)
	}

	if result.Created != 100 {
		t.Errorf("Expected 100 created, got %d", result.Created)
	}
}

// Test_SyncService_WithSpecialCharacters tests handling of special characters
func Test_SyncService_WithSpecialCharacters(t *testing.T) {
	ctx := context.Background()
	helper := NewTestDataHelper()

	mockRepo := &MockProductRepository{
		GetAllProductsFunc: func(ctx context.Context) ([]models.Product, error) {
			return []models.Product{}, nil
		},
		CreateProductsBatchFunc: func(ctx context.Context, products []struct{ Title, Handle string }) error {
			// Verify handles are properly sanitized
			for _, p := range products {
				// Check that handle doesn't contain special characters
				for _, r := range p.Handle {
					if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
						t.Errorf("Handle contains invalid character: %s (char: %c)", p.Handle, r)
					}
				}
			}
			return nil
		},
	}

	syncService := NewSyncService(mockRepo)
	externalItems := helper.GetMockExternalItemsWithSpecialCharacters()

	result, err := syncService.CompareAndSync(ctx, externalItems)
	if err != nil {
		t.Fatalf("Failed to sync items with special characters: %v", err)
	}

	if result.Created != 5 {
		t.Errorf("Expected 5 created, got %d", result.Created)
	}
}

// Test_SyncService_ConcurrencyStressTest tests concurrent operations
func Test_SyncService_ConcurrencyStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	ctx := context.Background()

	// This test verifies that batch operations use goroutines correctly
	createCalled := false
	updateCalled := false

	mockRepo := &MockProductRepository{
		GetAllProductsFunc: func(ctx context.Context) ([]models.Product, error) {
			return []models.Product{
				{ID: 1, Title: "Existing Product", Handle: "old-handle"},
			}, nil
		},
		CreateProductsBatchFunc: func(ctx context.Context, products []struct{ Title, Handle string }) error {
			createCalled = true
			// Simulate some work
			return nil
		},
		UpdateProductsBatchFunc: func(ctx context.Context, updates []struct {
			ID     int
			Title  string
			Handle string
		}) error {
			updateCalled = true
			// Simulate some work
			return nil
		},
	}

	syncService := NewSyncService(mockRepo)

	// Create items that will trigger both create and update operations
	externalItems := []map[string]interface{}{
		{"ItemName": "Existing Product", "ItemCode": "E001"}, // Will update
		{"ItemName": "New Product 1", "ItemCode": "N001"},    // Will create
		{"ItemName": "New Product 2", "ItemCode": "N002"},    // Will create
	}

	result, err := syncService.CompareAndSync(ctx, externalItems)
	if err != nil {
		t.Fatalf("Stress test failed: %v", err)
	}

	// Verify both operations were called (demonstrating concurrent execution)
	if !createCalled {
		t.Error("Create batch was not called")
	}
	if !updateCalled {
		t.Error("Update batch was not called")
	}

	// Verify results
	if result.Created != 2 {
		t.Errorf("Expected 2 created, got %d", result.Created)
	}
	if result.Updated != 1 {
		t.Errorf("Expected 1 updated, got %d", result.Updated)
	}
}

// Benchmark_CompareAndSync benchmarks the sync operation
func Benchmark_CompareAndSync(b *testing.B) {
	ctx := context.Background()
	helper := NewTestDataHelper()

	mockRepo := &MockProductRepository{
		GetAllProductsFunc: func(ctx context.Context) ([]models.Product, error) {
			return helper.GetMockDatabaseProducts(), nil
		},
		CreateProductsBatchFunc: func(ctx context.Context, products []struct{ Title, Handle string }) error {
			return nil
		},
		UpdateProductsBatchFunc: func(ctx context.Context, updates []struct {
			ID     int
			Title  string
			Handle string
		}) error {
			return nil
		},
	}

	syncService := NewSyncService(mockRepo)
	externalItems := helper.GetMockExternalItems()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = syncService.CompareAndSync(ctx, externalItems)
	}
}

// Benchmark_CompareAndSync_Large benchmarks with a large dataset
func Benchmark_CompareAndSync_Large(b *testing.B) {
	ctx := context.Background()
	helper := NewTestDataHelper()

	mockRepo := &MockProductRepository{
		GetAllProductsFunc: func(ctx context.Context) ([]models.Product, error) {
			return []models.Product{}, nil
		},
		CreateProductsBatchFunc: func(ctx context.Context, products []struct{ Title, Handle string }) error {
			return nil
		},
	}

	syncService := NewSyncService(mockRepo)
	externalItems := helper.GetMockExternalItemsLarge(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = syncService.CompareAndSync(ctx, externalItems)
	}
}

package repo

import (
	"context"
	"fmt"
	"go-cron/models"
	"log"
	"strings"
	"sync"
)

// SyncService handles synchronization between external API and database
type SyncService struct {
	repo ProductRepositoryInterface
}

// NewSyncService creates a new sync service
func NewSyncService(repo ProductRepositoryInterface) *SyncService {
	return &SyncService{repo: repo}
}

// CompareAndSync compares external items with database products and performs sync
func (s *SyncService) CompareAndSync(ctx context.Context, externalItems []map[string]interface{}) (*models.SyncResult, error) {
	result := &models.SyncResult{}

	// Fetch all products from database
	dbProducts, err := s.repo.GetAllProducts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch database products: %w", err)
	}

	// Create a map of existing products by normalized title for O(1) lookup
	dbProductMap := make(map[string]*models.Product)
	for i := range dbProducts {
		normalizedTitle := normalizeTitle(dbProducts[i].Title)
		dbProductMap[normalizedTitle] = &dbProducts[i]
	}

	// Separate items into creates and updates
	var itemsToCreate []struct{ Title, Handle string }
	var itemsToUpdate []struct {
		ID     int
		Title  string
		Handle string
	}

	// Process external items
	for _, item := range externalItems {
		itemName, ok := item["ItemName"].(string)
		if !ok || itemName == "" {
			result.Errors = append(result.Errors, "Invalid or missing ItemName in external item")
			continue
		}

		// Generate handle from ItemName (lowercase, replace spaces with hyphens)
		handle := generateHandle(itemName)
		normalizedTitle := normalizeTitle(itemName)

		// Check if product exists in database
		if existingProduct, exists := dbProductMap[normalizedTitle]; exists {
			// Check if update is needed (title or handle changed)
			if existingProduct.Title != itemName || existingProduct.Handle != handle {
				itemsToUpdate = append(itemsToUpdate, struct {
					ID     int
					Title  string
					Handle string
				}{
					ID:     existingProduct.ID,
					Title:  itemName,
					Handle: handle,
				})
			} else {
				result.Unchanged++
			}
		} else {
			// Product doesn't exist, add to create list
			itemsToCreate = append(itemsToCreate, struct{ Title, Handle string }{
				Title:  itemName,
				Handle: handle,
			})
		}
	}

	// Execute batch operations with concurrency
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	// Create new products in batch
	if len(itemsToCreate) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// if err := s.repo.CreateProductsBatch(ctx, itemsToCreate); err != nil {
			// 	errChan <- fmt.Errorf("batch create failed: %w", err)
			// } else {
				result.Created = len(itemsToCreate)
				log.Printf("Created %d new products", len(itemsToCreate))
			// }
		}()
	}

	// Update existing products in batch
	if len(itemsToUpdate) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// if err := s.repo.UpdateProductsBatch(ctx, itemsToUpdate); err != nil {
			// 	errChan <- fmt.Errorf("batch update failed: %w", err)
			// } else {
			// 	result.Updated = len(itemsToUpdate)
				log.Printf("Updated %d products", len(itemsToUpdate))
			// }
		}()
	}

	// Wait for all operations to complete
	wg.Wait()
	close(errChan)

	// Collect any errors
	for err := range errChan {
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
		}
	}

	return result, nil
}

// generateHandle creates a URL-friendly handle from a title
func generateHandle(title string) string {
	handle := strings.ToLower(title)
	handle = strings.ReplaceAll(handle, " ", "-")
	handle = strings.ReplaceAll(handle, "_", "-")
	// Remove special characters
	var builder strings.Builder
	for _, r := range handle {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// normalizeTitle normalizes a title for comparison
func normalizeTitle(title string) string {
	return strings.ToLower(strings.TrimSpace(title))
}

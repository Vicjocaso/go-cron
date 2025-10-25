package repo

import "go-cron/models"

// TestDataHelper provides mock data for testing
type TestDataHelper struct{}

// NewTestDataHelper creates a new test data helper
func NewTestDataHelper() *TestDataHelper {
	return &TestDataHelper{}
}

// GetMockExternalItems returns mock external API items
func (h *TestDataHelper) GetMockExternalItems() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"ItemCode":       "ITEM001",
			"ItemName":       "Premium Coffee Beans",
			"ItemsGroupCode": 100,
		},
		{
			"ItemCode":       "ITEM002",
			"ItemName":       "Organic Green Tea",
			"ItemsGroupCode": 101,
		},
		{
			"ItemCode":       "ITEM003",
			"ItemName":       "Dark Chocolate Bar",
			"ItemsGroupCode": 121,
		},
		{
			"ItemCode":       "ITEM004",
			"ItemName":       "Vanilla Extract",
			"ItemsGroupCode": 100,
		},
		{
			"ItemCode":       "ITEM005",
			"ItemName":       "Honey Jar 500g",
			"ItemsGroupCode": 101,
		},
	}
}

// GetMockDatabaseProducts returns mock database products
func (h *TestDataHelper) GetMockDatabaseProducts() []models.Product {
	return []models.Product{
		{
			ID:     1,
			Title:  "Premium Coffee Beans",
			Handle: "premium-coffee-beans",
		},
		{
			ID:     2,
			Title:  "Organic Green Tea",
			Handle: "organic-green-tea",
		},
		{
			ID:     3,
			Title:  "Old Product Name", // This will be updated
			Handle: "old-product-name",
		},
	}
}

// GetMockExternalItemsLarge returns a large set of mock items for performance testing
func (h *TestDataHelper) GetMockExternalItemsLarge(count int) []map[string]interface{} {
	items := make([]map[string]interface{}, count)
	for i := 0; i < count; i++ {
		items[i] = map[string]interface{}{
			"ItemCode":       formatItemCode(i),
			"ItemName":       formatItemName(i),
			"ItemsGroupCode": 100 + (i % 3),
		}
	}
	return items
}

// Helper functions for generating test data
func formatItemCode(i int) string {
	return "ITEM" + padLeft(i, 6)
}

func formatItemName(i int) string {
	return "Product " + padLeft(i, 6)
}

func padLeft(num int, length int) string {
	s := ""
	for i := 0; i < length; i++ {
		s += "0"
	}
	s += string(rune('0' + num))
	return s[len(s)-length:]
}

// GetMockExternalItemsWithInvalidData returns items with various invalid data
func (h *TestDataHelper) GetMockExternalItemsWithInvalidData() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"ItemCode":       "VALID001",
			"ItemName":       "Valid Product 1",
			"ItemsGroupCode": 100,
		},
		{
			"ItemCode":       "INVALID001",
			"ItemName":       "", // Invalid: empty name
			"ItemsGroupCode": 100,
		},
		{
			"ItemCode": "INVALID002",
			// Missing ItemName
			"ItemsGroupCode": 100,
		},
		{
			"ItemCode":       "INVALID003",
			"ItemName":       123, // Invalid: wrong type
			"ItemsGroupCode": 100,
		},
		{
			"ItemCode":       "VALID002",
			"ItemName":       "Valid Product 2",
			"ItemsGroupCode": 101,
		},
	}
}

// GetMockExternalItemsWithSpecialCharacters returns items with special characters
func (h *TestDataHelper) GetMockExternalItemsWithSpecialCharacters() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"ItemCode": "SPECIAL001",
			"ItemName": "Café Latté",
		},
		{
			"ItemCode": "SPECIAL002",
			"ItemName": "Product @ #$% Special & Chars",
		},
		{
			"ItemCode": "SPECIAL003",
			"ItemName": "Product_With_Underscores",
		},
		{
			"ItemCode": "SPECIAL004",
			"ItemName": "Product   With   Multiple   Spaces",
		},
		{
			"ItemCode": "SPECIAL005",
			"ItemName": "123 Numeric Product 456",
		},
	}
}

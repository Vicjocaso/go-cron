# Testing Guide

## Overview

This directory contains comprehensive tests for the repository and sync service layers using **modern Go testing practices** with interface-based mocking.

## Why This Approach?

Instead of using outdated mocking libraries like `DATA-DOG/go-sqlmock` (last updated over a year ago), we use:

1. **Interface-based mocking** - Clean, maintainable, no external dependencies
2. **Go's standard testing library** - Built-in, always up-to-date
3. **Dependency injection** - Easy to swap implementations for testing

## Test Structure

```
repo/
├── interfaces.go      - Repository interface definition
├── product.go         - Repository implementation
├── sync.go           - Sync service implementation
├── sync_test.go      - Comprehensive test suite
└── testdata.go       - Mock data helpers
```

## Running Tests

### Run All Tests

```bash
go test ./repo/... -v
```

### Run Specific Test

```bash
go test ./repo/... -v -run Test_SyncService_CompareAndSync_NewItems
```

### Run Tests with Coverage

```bash
go test ./repo/... -cover
```

### Generate Coverage Report

```bash
go test ./repo/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Run Tests with Race Detector (checks concurrency issues)

```bash
go test ./repo/... -race
```

## Test Cases Covered

### ✅ Sync Service Tests

#### 1. **Test_SyncService_CompareAndSync_NewItems**

- Tests creating new items that don't exist in the database
- **Scenario**: Empty database, 3 new external items
- **Expected**: 3 created, 0 updated, 0 unchanged

#### 2. **Test_SyncService_CompareAndSync_UpdateExisting**

- Tests updating existing items with new data
- **Scenario**: 2 existing products with old handles
- **Expected**: 0 created, 2 updated, 0 unchanged

#### 3. **Test_SyncService_CompareAndSync_UnchangedItems**

- Tests items that match exactly (no updates needed)
- **Scenario**: Database matches external API perfectly
- **Expected**: 0 created, 0 updated, 2 unchanged

#### 4. **Test_SyncService_CompareAndSync_MixedScenario**

- Tests realistic mixed scenario
- **Scenario**: 1 unchanged, 1 updated, 2 new items
- **Expected**: 2 created, 1 updated, 1 unchanged

#### 5. **Test_SyncService_CompareAndSync_InvalidData**

- Tests handling of invalid external data
- **Scenario**: Mix of valid and invalid items (empty names, missing fields, wrong types)
- **Expected**: Errors logged, valid items processed

#### 6. **Test_SyncService_CompareAndSync_CaseInsensitiveMatching**

- Tests case-insensitive title matching
- **Scenario**: External items with different casing than database
- **Expected**: Items matched correctly regardless of case

### ✅ Utility Function Tests

#### 7. **Test_generateHandle**

- Tests URL-friendly handle generation
- **Cases**:
  - Simple text → `simple-product`
  - Mixed case → `product-with-caps`
  - Underscores → `product-with-underscores`
  - Special characters → Removed
  - Multiple spaces → Multiple hyphens
  - Accented characters → Removed

#### 8. **Test_normalizeTitle**

- Tests title normalization for comparison
- **Cases**:
  - Trim spaces
  - Lowercase conversion
  - Edge cases (empty, all spaces)

## Mock Repository Pattern

The tests use a custom mock implementation:

```go
type MockProductRepository struct {
    GetAllProductsFunc      func(ctx context.Context) ([]models.Product, error)
    CreateProductsBatchFunc func(ctx context.Context, products []struct{ Title, Handle string }) error
    // ... other functions
}
```

**Benefits**:

- ✅ No external dependencies
- ✅ Full control over mock behavior
- ✅ Easy to customize per test
- ✅ Type-safe
- ✅ No generated code

## Using Mock Data

The `testdata.go` file provides reusable mock data:

```go
helper := NewTestDataHelper()

// Get mock external items
externalItems := helper.GetMockExternalItems()

// Get mock database products
dbProducts := helper.GetMockDatabaseProducts()

// Get large dataset for performance testing
largeDataset := helper.GetMockExternalItemsLarge(1000)
```

## Writing New Tests

### 1. Create a Mock Repository

```go
mockRepo := &MockProductRepository{
    GetAllProductsFunc: func(ctx context.Context) ([]models.Product, error) {
        return []models.Product{
            {ID: 1, Title: "Test Product", Handle: "test-product"},
        }, nil
    },
    CreateProductsBatchFunc: func(ctx context.Context, products []struct{ Title, Handle string }) error {
        // Add assertions here
        return nil
    },
}
```

### 2. Create Sync Service with Mock

```go
syncService := NewSyncService(mockRepo)
```

### 3. Prepare Test Data

```go
externalItems := []map[string]interface{}{
    {"ItemName": "Test Product", "ItemCode": "TEST001"},
}
```

### 4. Execute and Assert

```go
result, err := syncService.CompareAndSync(ctx, externalItems)
if err != nil {
    t.Fatalf("CompareAndSync failed: %v", err)
}

if result.Created != 1 {
    t.Errorf("Expected 1 created, got %d", result.Created)
}
```

## Test Coverage

Current test coverage includes:

- ✅ **Happy Path Scenarios**: Create, update, unchanged
- ✅ **Edge Cases**: Empty data, invalid data, special characters
- ✅ **Concurrency**: Tests use goroutines (run with `-race` flag)
- ✅ **Error Handling**: Invalid input, database errors
- ✅ **Business Logic**: Case-insensitive matching, handle generation
- ✅ **Performance**: Batch operations, O(1) lookups

## Integration Testing (Future)

For integration tests with a real database, consider:

1. **testcontainers-go** - Spin up PostgreSQL in Docker
2. **In-memory SQLite** - For lightweight integration tests
3. **Test database** - Dedicated test PostgreSQL instance

Example with testcontainers (when needed):

```go
// Future implementation
func setupTestDatabase(t *testing.T) *sql.DB {
    // Start PostgreSQL container
    // Run migrations
    // Return DB connection
}
```

## CI/CD Integration

Add to your CI pipeline:

```yaml
- name: Run Tests
  run: |
    go test ./... -v -race -coverprofile=coverage.out
    go tool cover -func=coverage.out
```

## Best Practices Applied

1. ✅ **Table-Driven Tests** - For testing multiple cases
2. ✅ **Interface-Based Mocking** - No external dependencies
3. ✅ **Descriptive Test Names** - Clear what's being tested
4. ✅ **Isolated Tests** - Each test is independent
5. ✅ **Mock Data Helpers** - Reusable test data
6. ✅ **Context Usage** - Proper timeout handling
7. ✅ **Error Assertions** - Both success and failure paths

## Troubleshooting

### Tests are slow

- Run specific tests with `-run` flag
- Use `-short` flag to skip long-running tests
- Consider parallel execution with `t.Parallel()`

### Race condition warnings

- Run with `-race` flag to detect
- Fix by adding proper synchronization (mutexes, channels)

### Flaky tests

- Ensure tests are isolated
- Avoid time-dependent logic
- Use mock time if needed

## Example Test Run Output

```
=== RUN   Test_SyncService_CompareAndSync_NewItems
    Created 3 new products
--- PASS: Test_SyncService_CompareAndSync_NewItems (0.00s)
=== RUN   Test_SyncService_CompareAndSync_UpdateExisting
    Updated 2 products
--- PASS: Test_SyncService_CompareAndSync_UpdateExisting (0.00s)
=== RUN   Test_SyncService_CompareAndSync_MixedScenario
    Updated 1 products
    Created 2 new products
--- PASS: Test_SyncService_CompareAndSync_MixedScenario (0.00s)
PASS
ok      go-cron/repo    0.286s
```

## Contributing

When adding new features:

1. Write tests first (TDD)
2. Ensure all existing tests still pass
3. Add new test cases for edge cases
4. Update this documentation
5. Run with `-race` flag before committing

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Go Test Coverage](https://go.dev/blog/cover)
- [Table-Driven Tests](https://go.dev/wiki/TableDrivenTests)
- [Advanced Testing with Go](https://www.youtube.com/watch?v=8hQG7QlcLBk)

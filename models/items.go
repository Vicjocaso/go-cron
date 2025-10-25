package models

type ItemsResponse struct {
	ODataMetadata string                   `json:"odata.metadata"`
	ODataNextLink string                   `json:"odata.nextLink"`
	Value         []map[string]interface{} `json:"value"`
}

// Product represents a product in the database
type Product struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Handle string `json:"handle"`
}

// ExternalItem represents an item from the external API
type ExternalItem struct {
	ItemCode       string `json:"ItemCode"`
	ItemName       string `json:"ItemName"`
	ItemsGroupCode int    `json:"ItemsGroupCode"`
}

// SyncResult contains statistics about the sync operation
type SyncResult struct {
	Created   int      `json:"created"`
	Updated   int      `json:"updated"`
	Unchanged int      `json:"unchanged"`
	Errors    []string `json:"errors,omitempty"`
}

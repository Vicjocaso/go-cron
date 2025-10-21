package models

type ItemsResponse struct {
	ODataMetadata string                   `json:"odata.metadata"`
	ODataNextLink string                   `json:"odata.nextLink"`
	Value         []map[string]interface{} `json:"value"`
}

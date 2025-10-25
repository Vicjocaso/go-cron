package handler

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-cron/config"
	"go-cron/models"
	"go-cron/repo"
	"go-cron/utils"
)

// init function runs before main and is a great place to set up the DB connection.
func init() {
	utils.InitDB(config.LoadConfig())
}

// PageJob represents a page fetching job
type PageJob struct {
	Skip int
	Top  int
}

// PageResult represents the result of fetching a page
type PageResult struct {
	Items []map[string]interface{}
	Skip  int
	Err   error
}

func Handler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	// --- 1. Security Check ---
	config := config.LoadConfig()
	authHeader := r.Header.Get("authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") || strings.TrimPrefix(authHeader, "Bearer ") != config.Auth.CRONSecret {
		log.Println("Unauthorized access attempt.")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Initialize repository and sync service
	db := utils.GetDB()
	productRepo := repo.NewProductRepository(db)
	syncService := repo.NewSyncService(productRepo)

	// Step 1: Login and get session
	log.Println("Logging in to external API...")
	sessionID, err := login(config)
	if err != nil {
		log.Printf("Login failed: %v\n", err)
		http.Error(w, fmt.Sprintf("Login failed: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("Logged in successfully with session: %s\n", sessionID)

	// Ensure logout happens at the end
	defer func() {
		if err := logout(config.ExternalAPI.ExternalAPIURL, sessionID); err != nil {
			log.Printf("Logout failed: %v\n", err)
		} else {
			log.Println("Logged out successfully")
		}
	}()

	// Step 2: Get the total count of items
	log.Println("Fetching item count from external API...")
	count, err := getItemCount(config, sessionID)
	if err != nil {
		log.Printf("Failed to get item count: %v\n", err)
		http.Error(w, fmt.Sprintf("Failed to get item count: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("Total count of items: %d\n", count)

	// Step 3: Fetch all items concurrently using worker pool
	pageSize := 20
	numWorkers := 2 // Number of concurrent workers

	log.Printf("Starting concurrent fetch with %d workers...\n", numWorkers)
	allItems, err := fetchAllItemsConcurrently(ctx, config, sessionID, count, pageSize, numWorkers)
	if err != nil {
		log.Printf("Failed to fetch items: %v\n", err)
		http.Error(w, fmt.Sprintf("Failed to fetch items: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("Successfully fetched %d items from external API\n", len(allItems))

	// Step 4: Sync with database
	log.Println("Starting database synchronization...")
	syncResult, err := syncService.CompareAndSync(ctx, allItems)
	if err != nil {
		log.Printf("Sync failed: %v\n", err)
		http.Error(w, fmt.Sprintf("Sync failed: %v", err), http.StatusInternalServerError)
		return
	}

	duration := time.Since(startTime)
	log.Printf("Sync completed in %v - Created: %d, Updated: %d, Unchanged: %d\n",
		duration, syncResult.Created, syncResult.Updated, syncResult.Unchanged)

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":      "Successfully synchronized data from external API",
		"totalItems":   count,
		"itemsFetched": len(allItems),
		"syncResult":   syncResult,
		"duration":     duration.String(),
	})
}

// fetchAllItemsConcurrently fetches all items from external API using a worker pool pattern
func fetchAllItemsConcurrently(ctx context.Context, config *models.AppConfig, sessionID string, totalCount, pageSize, numWorkers int) ([]map[string]interface{}, error) {
	// Create job channel and result channel
	jobs := make(chan PageJob, numWorkers*2)
	results := make(chan PageResult, numWorkers*2)

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			worker(ctx, workerID, config, sessionID, pageSize, jobs, results)
		}(i)
	}

	// Send jobs to workers
	go func() {
		for skip := 0; skip < totalCount; skip += pageSize {
			select {
			case jobs <- PageJob{Skip: skip, Top: pageSize}:
			case <-ctx.Done():
				close(jobs)
				return
			}
		}
		close(jobs)
	}()

	// Collect results in a separate goroutine
	allResults := make([]PageResult, 0)
	var resultWg sync.WaitGroup
	resultWg.Add(1)
	go func() {
		defer resultWg.Done()
		for result := range results {
			allResults = append(allResults, result)
		}
	}()

	// Wait for all workers to finish
	wg.Wait()
	close(results)

	// Wait for result collection to finish
	resultWg.Wait()

	// Check for errors and combine all items
	var allItems []map[string]interface{}
	for _, result := range allResults {
		if result.Err != nil {
			return nil, fmt.Errorf("error fetching page at skip %d: %w", result.Skip, result.Err)
		}
		allItems = append(allItems, result.Items...)
	}

	return allItems, nil
}

// worker is a worker goroutine that fetches pages from the external API
func worker(ctx context.Context, workerID int, config *models.AppConfig, sessionID string, pageSize int, jobs <-chan PageJob, results chan<- PageResult) {
	log.Printf("Worker %d started\n", workerID)

	for job := range jobs {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d cancelled\n", workerID)
			return
		default:
			log.Printf("Worker %d fetching page at skip=%d\n", workerID, job.Skip)
			items, err := fetchItemsPage(config, sessionID, job.Top, job.Skip)

			result := PageResult{
				Items: items,
				Skip:  job.Skip,
				Err:   err,
			}

			select {
			case results <- result:
				if err == nil {
					log.Printf("Worker %d completed page at skip=%d (%d items)\n", workerID, job.Skip, len(items))
				} else {
					log.Printf("Worker %d error at skip=%d: %v\n", workerID, job.Skip, err)
				}
			case <-ctx.Done():
				log.Printf("Worker %d cancelled while sending result\n", workerID)
				return
			}
		}
	}

	log.Printf("Worker %d finished\n", workerID)
}

func getItemCount(config *models.AppConfig, sessionID string) (int, error) {
	baseURL := config.ExternalAPI.ExternalAPIURL
	u, err := url.Parse(baseURL + config.ExternalAPI.ItemsURL + "/$count?")
	if err != nil {
		return 0, fmt.Errorf("failed to parse base URL: %v", err)
	}

	params := url.Values{}
	params.Add("$select", "ItemCode,ItemName,ItemsGroupCode")
	params.Add("$filter", "ItemsGroupCode eq 100 or ItemsGroupCode eq 101 or ItemsGroupCode eq 121")
	params.Add("$orderby", "ItemCode")

	u.RawQuery = params.Encode()

	jar, err := cookiejar.New(nil)
	if err != nil {
		return 0, err
	}
	jar.SetCookies(u, []*http.Cookie{{Name: "B1SESSION", Value: sessionID}})

	client := &http.Client{
		Jar: jar,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "text/plain")

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("count fetch failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	count, err := strconv.Atoi(strings.TrimSpace(string(body)))
	if err != nil {
		return 0, fmt.Errorf("failed to parse count: %v", err)
	}

	return count, nil
}

func login(config *models.AppConfig) (string, error) {
	loginURL := config.ExternalAPI.ExternalAPIURL + config.ExternalAPI.LoginURL
	reqBody := models.Credentials{
		CompanyDB: config.ExternalAuth.CompanyDB,
		UserName:  config.ExternalAuth.UserName,
		Password:  config.ExternalAuth.Password,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", loginURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	var loginResp models.LoginResponse
	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	if err != nil {
		return "", err
	}

	return loginResp.SessionID, nil
}

func fetchItemsPage(config *models.AppConfig, sessionID string, top, skip int) ([]map[string]interface{}, error) {
	u, err := url.Parse(config.ExternalAPI.ExternalAPIURL + config.ExternalAPI.ItemsURL + "?")
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %v", err)
	}

	params := url.Values{}
	params.Add("$select", "ItemCode,ItemName,ItemsGroupCode")
	params.Add("$filter", "ItemsGroupCode eq 100 or ItemsGroupCode eq 101 or ItemsGroupCode eq 121")
	params.Add("$orderby", "ItemCode")
	params.Add("$top", strconv.Itoa(top))
	params.Add("$skip", strconv.Itoa(skip))

	u.RawQuery = params.Encode()

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	jar.SetCookies(u, []*http.Cookie{{Name: "B1SESSION", Value: sessionID}})

	client := &http.Client{
		Jar: jar,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetch failed with status %d: %s", resp.StatusCode, string(body))
	}

	var itemsResp models.ItemsResponse
	err = json.NewDecoder(resp.Body).Decode(&itemsResp)
	if err != nil {
		return nil, err
	}

	return itemsResp.Value, nil
}

func logout(baseURL, sessionID string) error {
	logoutURL := baseURL + "/Logout"

	req, err := http.NewRequest("POST", logoutURL, nil)
	if err != nil {
		return err
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}
	u, _ := url.Parse(baseURL)
	jar.SetCookies(u, []*http.Cookie{{Name: "B1SESSION", Value: sessionID}})

	client := &http.Client{
		Jar: jar,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("logout failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

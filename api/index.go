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
	"os"
	"strconv"
	"strings"

	"go-cron/config"
	"go-cron/models"
	"go-cron/utils"

	"github.com/jackc/pgx/v5/pgxpool"
)

// init function runs before main and is a great place to set up the DB connection.
func init() {
	utils.InitDB(config.LoadConfig())
}

// InsertItem inserts a new item into the database.
func InsertItem(ctx context.Context, pool *pgxpool.Pool, title string) (int, error) {
	// The SQL statement for insertion. Using RETURNING id is efficient
	// as it returns the new item's ID without a second query.
	sqlStatement := `
		INSERT INTO products (title)
		VALUES ($1)
		RETURNING id`

	var newID int
	// pool.QueryRow is used for queries that are expected to return a single row.
	//.Scan() then reads the returned 'id' into our newID variable.
	err := pool.QueryRow(ctx, sqlStatement, title).Scan(&newID)
	if err != nil {
		// It's good practice to wrap errors for more context.
		return 0, fmt.Errorf("unable to insert item: %w", err)
	}

	return newID, nil
}

func Handler(w http.ResponseWriter, r *http.Request) {
	// --- 1. Security Check ---
	config := config.LoadConfig()
	authHeader := r.Header.Get("authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") || strings.TrimPrefix(authHeader, "Bearer ") != config.Auth.CRONSecret {
		log.Println("Unauthorized access attempt.")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Step 1: Login and get session
	sessionID, err := login(config)
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}
	fmt.Printf("Logged in with session: %s\n", sessionID)

	// Step 2: Get the total count of items
	count, err := getItemCount(config, sessionID)
	if err != nil {
		fmt.Printf("Failed to get item count: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Total count of items: %d\n", count)

	// Step 3: Fetch all items by looping through pages in batches of 100
	pageSize := 20
	var allItems []map[string]interface{}
	for skip := 0; skip < count; skip += pageSize {
		pageItems, err := fetchItemsPage(config, sessionID, pageSize, skip)
		if err != nil {
			fmt.Printf("Failed to fetch page at skip %d: %v\n", skip, err)
			os.Exit(1)
		}

		if len(pageItems) == 0 {
			break
		}

		allItems = append(allItems, pageItems...)

		// Ensure the connection pool is closed when the application exits.
		// defer utils.DB.Close()

		// fmt.Println("Attempting to insert a new item...")
		// fmt.Println(pageItems[0]["ItemName"].(string))

		// insertedID, err := InsertItem(utils.DB, pageItems[0]["ItemName"].(string))
		// if err != nil {
		// 	log.Fatalf("Error inserting item: %v", err)
		// }
		// fmt.Println(insertedID)
		fmt.Printf("Fetched %d items (skip=%d)\n", len(pageItems), skip)
	}

	err = logout(config.ExternalAPI.ExternalAPIURL, sessionID)
	if err != nil {
		fmt.Printf("Logout failed: %v\n", err)
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}
	fmt.Printf("Logged out successfully\n")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":      "Success Fetching Data from SAP",
		"totalItems":   strconv.Itoa(count),
		"itemsFetched": allItems,
	})
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

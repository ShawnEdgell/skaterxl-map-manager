package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

var Logger *log.Logger = log.Default()

type DownloadInfo struct {
	BinaryURL   string `json:"binary_url"`
	DateExpires int64  `json:"date_expires"`
}

type Modfile struct {
	ID        int          `json:"id"`
	Filename  string       `json:"filename"`
	Version   string       `json:"version"`
	Filesize  int          `json:"filesize"`
	Download  DownloadInfo `json:"download"`
}

type Map struct {
	ID          int    `json:"id"`
	GameID      int    `json:"game_id"`
	Name        string `json:"name"`
	NameID      string `json:"name_id"`
	Summary     string `json:"summary"`
	DescriptionPlaintext string `json:"description_plaintext"`
	ProfileURL  string `json:"profile_url"`
	SubmittedBy struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		ProfileURL string `json:"profile_url"`
	} `json:"submitted_by"`
	DateAdded   int64 `json:"date_added"`
	DateUpdated int64 `json:"date_updated"`
	DateLive    int64 `json:"date_live"`
	Logo        struct {
		Filename    string `json:"filename"`
		Original    string `json:"original"`
		Thumb320x180 string `json:"thumb_320x180"`
	} `json:"logo"`
	Modfile Modfile `json:"modfile"`
	Tags    []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"tags"`
	Stats struct {
		DownloadsTotal   int    `json:"downloads_total"`
		SubscribersTotal int    `json:"subscribers_total"`
		RatingsPositive  int    `json:"ratings_positive"`
		RatingsNegative  int    `json:"ratings_negative"`
		RatingsDisplayText string `json:"ratings_display_text"`
	} `json:"stats"`
	Media struct {
		Images []struct {
			Filename string `json:"filename"`
			Original string `json:"original"`
		} `json:"images"`
	} `json:"media"`
}

type APIResponse struct {
	ItemType    string    `json:"itemType"`
	LastUpdated time.Time `json:"lastUpdated"`
	Count       int       `json:"count"`
	Items       []Map     `json:"items"`
}

const APIEndpoint = "https://api.skatebit.app/api/v1/skaterxl/maps"

func FetchMaps() ([]Map, error) {
	Logger.Println("Fetching maps from API:", APIEndpoint)
	resp, err := http.Get(APIEndpoint)
	if err != nil {
		Logger.Printf("Error during HTTP GET to %s: %v", APIEndpoint, err)
		return nil, fmt.Errorf("error fetching maps: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		Logger.Printf("API returned non-OK status: %s. Response Body (first 500 chars): %s", resp.Status, string(bodyBytes)[:min(len(bodyBytes), 500)])
		return nil, fmt.Errorf("API returned non-OK status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		Logger.Printf("Error reading response body: %v", err)
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var apiResponse APIResponse
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		Logger.Printf("Error unmarshaling JSON: %v. JSON Body (first 500 chars): %s", err, string(body)[:min(len(body), 500)])
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	Logger.Printf("Successfully fetched %d maps from API.", apiResponse.Count)
	return apiResponse.Items, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
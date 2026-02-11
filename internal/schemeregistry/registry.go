package schemeregistry

import (
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	json "github.com/goccy/go-json"
)

var (
	registryURL string
	cache       sync.Map
	client      *http.Client
)

const defaultAccrualRate = 0.02

func init() {
	registryURL = os.Getenv("SCHEME_REGISTRY_URL")
	if registryURL != "" {
		client = &http.Client{
			Timeout: 2 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		}
	}
}

type schemeResponse struct {
	SchemeID    string  `json:"scheme_id"`
	AccrualRate float64 `json:"accrual_rate"`
}

// GetAccrualRates fetches accrual rates for the given scheme IDs.
// Uses caching and concurrent fetching. Falls back to 0.02 on error.
func GetAccrualRates(schemeIDs []string) map[string]float64 {
	result := make(map[string]float64, len(schemeIDs))

	if registryURL == "" {
		for _, id := range schemeIDs {
			result[id] = defaultAccrualRate
		}
		return result
	}

	var toFetch []string
	for _, id := range schemeIDs {
		if rate, ok := cache.Load(id); ok {
			result[id] = rate.(float64)
		} else {
			toFetch = append(toFetch, id)
		}
	}

	if len(toFetch) == 0 {
		return result
	}

	if len(toFetch) == 1 {
		rate := fetchRate(toFetch[0])
		cache.Store(toFetch[0], rate)
		result[toFetch[0]] = rate
		return result
	}

	// Fetch concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, id := range toFetch {
		wg.Add(1)
		go func(schemeID string) {
			defer wg.Done()
			rate := fetchRate(schemeID)
			cache.Store(schemeID, rate)
			mu.Lock()
			result[schemeID] = rate
			mu.Unlock()
		}(id)
	}
	wg.Wait()

	return result
}

func fetchRate(schemeID string) float64 {
	resp, err := client.Get(registryURL + "/schemes/" + schemeID)
	if err != nil {
		return defaultAccrualRate
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return defaultAccrualRate
	}

	var sr schemeResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return defaultAccrualRate
	}
	return sr.AccrualRate
}

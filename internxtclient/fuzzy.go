package internxtclient

import (
	"context"
	"net/http"
	"net/url"
	"path"
	"strconv"
)

type FuzzyService struct {
	client *Client
}

type SearchResult struct {
	ID         string  `json:"id"`
	ItemID     string  `json:"itemId"`
	ItemType   string  `json:"itemType"` // "file" or "folder"
	Name       string  `json:"name"`
	Rank       float64 `json:"rank"`
	Similarity float64 `json:"similarity"`
}

type SearchResponse struct {
	Data []SearchResult `json:"data"`
}

// FuzzySearch performs a fuzzy search with a given term and offset.
func (f *FuzzyService) FuzzySearch(ctx context.Context, term string, offset int) (*SearchResponse, error) {
	encodedTerm := url.PathEscape(term)
	endpoint := path.Join("fuzzy", encodedTerm)

	var result SearchResponse

	if resp, err := f.client.doRequestWithQuery(ctx, APITypeDrive, http.MethodGet, endpoint, map[string]string{"offset": strconv.Itoa(offset)}, nil, &result, nil); err != nil {
		return nil, f.client.GetError(endpoint, resp, err)
	}

	return &result, nil
}

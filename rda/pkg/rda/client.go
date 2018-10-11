package rda

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Client performs HTTP requests to retrieve items from RDA.
type Client interface {
	Get(url string) (*http.Response, error)
}

// FetchMetadata returns Metadata for the provided RDA graphID and nodeID.
func FetchMetadata(graphID, nodeID string, client Client) (*Metadata, error) {
	res, err := client.Get(fmt.Sprintf(rdaMetadataEnpoint, graphID, nodeID))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	md := Metadata{}
	json.NewDecoder(res.Body).Decode(&md)
	return &md, nil
}

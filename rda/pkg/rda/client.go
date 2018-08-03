package rda

import (
	"net/http"
)

// Client performs HTTP requests to retrieve items from RDA.
type Client interface {
	Get(url string) (*http.Response, error)
}

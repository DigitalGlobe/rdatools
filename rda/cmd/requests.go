package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/DigitalGlobe/rdatools/rda/pkg/rda"
)

func Metadata(graphID, nodeID string, config Config) *rda.Metadata {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://rda.geobigdata.io/v1/metadata/%s/%s/metadata.json", graphID, nodeID), nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", config.Token.AccessToken))
	res, err := client.Do(req)
	if err != nil {
		log.Fatalln(req, err)
	}
	defer res.Body.Close()

	md := rda.Metadata{}
	json.NewDecoder(res.Body).Decode(&md)
	return &md
}

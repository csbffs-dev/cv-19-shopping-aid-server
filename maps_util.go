package main

import (
	"fmt"
	"os"

	"googlemaps.github.io/maps"
)

// MapsClient returns a new client to Google Maps APIs
func MapsClient() (*maps.Client, error) {
	apiKey := os.Getenv("MAPS_CLIENT_API_KEY") // See GCP console for API key
	c, err := maps.NewClient(maps.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create maps client: %v", err)
	}
	return c, nil
}

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// DecodeReq is a helper for decoding JSON request bodies for handlers.
func DecodeReq(r io.ReadCloser, req interface{}) error {
	if err := json.NewDecoder(r).Decode(req); err != nil {
		return fmt.Errorf("failed to decode request body in json: %v", err)
	}
	return nil
}

// EncodeResp is a helper for encoding JSON response bodies for handlers.
func EncodeResp(w http.ResponseWriter, resp interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		return fmt.Errorf("failed to encode response in json: %v", err)
	}
	return nil
}

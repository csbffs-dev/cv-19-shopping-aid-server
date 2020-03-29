package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func DecodeReq(r io.ReadCloser, req interface{}) error {
	if err := json.NewDecoder(r).Decode(&req); err != nil {
		return fmt.Errorf("failed to decode request body in json: %v", err)
	}
	return nil
}

func EncodeResp(w http.ResponseWriter, resp interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(&resp); err != nil {
		return fmt.Errorf("failed to encode response in json: %v", err)
	}
	return nil
}

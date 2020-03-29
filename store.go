package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"cloud.google.com/go/datastore"
	"github.com/google/uuid"
)

type Store struct {
	StoreID string   `datastore:"storeID"`
	Name    string   `datastore:"name"`
	Addr    *Address `datastore:"addr"`
}

type Address struct {
	Street  string `datastore:"street"`
	City    string `datastore:"city"`
	State   string `datastore:"state"`
	ZipCode string `datastore:"zipCode"`
}

// QueryStores fetches the list of stores in storage.
func QueryStores(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	return http.StatusNotImplemented, fmt.Errorf("not supported yet")
}

type AddStoreReq struct {
	Name    string `json:"name"`
	Street  string `json:"street"`
	City    string `json:"city"`
	State   string `json:"state"`
	ZipCode string `json:"zip_code"`
}

type AddStoreResp struct {
	StoreID string `json:"store_id"`
}

func AddStore(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	var req AddStoreReq
	if err := DecodeReq(r.Body, &req); err != nil {
		return http.StatusBadRequest, err
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Street = strings.TrimSpace(req.Street)
	req.City = strings.TrimSpace(req.City)
	req.State = strings.TrimSpace(req.State)
	req.ZipCode = strings.TrimSpace(req.ZipCode)

	if err := validateAddStoreReq(req); err != nil {
		return http.StatusBadRequest, err
	}

	uid, err := uuid.NewRandom()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to generate store id: %v", err)
	}
	storeID := uid.String()

	store := &Store{
		StoreID: storeID,
		Name:    req.Name,
		Addr: &Address{
			Street:  req.Street,
			City:    req.City,
			State:   req.State,
			ZipCode: req.ZipCode,
		},
	}

	if err := createStoreInStorage(ctx, store); err != nil {
		return http.StatusInternalServerError, err
	}

	resp := &AddStoreResp{
		StoreID: storeID,
	}

	if err := EncodeResp(w, &resp); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func validateAddStoreReq(req AddStoreReq) error {
	if req.Name == "" {
		return fmt.Errorf("missing store name")
	}
	if req.Street == "" {
		return fmt.Errorf("missing street")
	}
	if req.City == "" {
		return fmt.Errorf("missing city")
	}
	if req.State == "" {
		return fmt.Errorf("missing state")
	}
	if req.ZipCode == "" {
		return fmt.Errorf("missing zip code")
	}
	return nil
}

func createStoreInStorage(ctx context.Context, st *Store) error {
	client, err := StorageClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	key := datastore.NameKey(StoreKind, st.StoreID, nil)
	_, err = client.Put(ctx, key, st)
	if err != nil {
		return fmt.Errorf("failed to add store in storage: %v", err)
	}
	return nil
}

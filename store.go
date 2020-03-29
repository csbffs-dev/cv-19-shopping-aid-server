package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"cloud.google.com/go/datastore"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

type Store struct {
	StoreID string   `datastore:"storeID" json:"store_id"`
	Name    string   `datastore:"name" json:"name"`
	Addr    *Address `datastore:"addr" json:"address"`
}

type Address struct {
	Street  string `datastore:"street" json:"street"`
	City    string `datastore:"city" json:"city"`
	State   string `datastore:"state" json:"state"`
	ZipCode string `datastore:"zipCode" json:"zip_code"`
}

type QueryStoresReq struct {
	UserID string `json:"user_id"`
}

type QueryStoresResp struct {
	Stores []Store `json:"stores"`
}

// QueryStores fetches the list of stores in storage.
func QueryStores(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	var req QueryStoresReq
	if err := DecodeReq(r.Body, &req); err != nil {
		return http.StatusBadRequest, err
	}

	if err := validateQueryStoresReq(req); err != nil {
		return http.StatusBadRequest, err
	}

	_, ok, err := GetUserInStorage(ctx, req.UserID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to check user creds: %v", err)
	}
	if !ok {
		return http.StatusForbidden, fmt.Errorf("user id is invalid: %q", req.UserID)
	}

	client, err := StorageClient(ctx)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	resp := &QueryStoresResp{}
	q := datastore.NewQuery(StoreKind)
	it := client.Run(ctx, q)
	for {
		var st Store
		_, err := it.Next(&st)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to query for all stores: %v", err)
		}
		resp.Stores = append(resp.Stores, st)
	}

	// TODO: Order resp.Stores based on distance between user zip code and store address

	if err := EncodeResp(w, &resp); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func validateQueryStoresReq(req QueryStoresReq) error {
	if req.UserID == "" {
		return fmt.Errorf("missing user id")
	}
	return nil
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

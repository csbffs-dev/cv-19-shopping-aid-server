package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/google/uuid"
)

// SetupUserReq represents request to SetupUser.
type SetupUserReq struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	ZipCode   string `json:"zip_code"`
}

// SetupUserResp represents response to SetupUser.
type SetupUserResp struct {
	UserID string `json:"user_id"`
}

// User represents the user entity in storage.
type User struct {
	UserID       string `datastore:"userID"`
	FirstName    string `datastore:"firstName"`
	LastName     string `datastore:"lastName"`
	ZipCode      string `datastore:"zipCode"`
	TimestampSec int64  `datastore:"timestampSec"`
}

// SetupUser sets up a user in storage.
func SetupUser(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	var req SetupUserReq
	if err := DecodeReq(r.Body, &req); err != nil {
		return http.StatusBadRequest, err
	}
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.ZipCode = strings.TrimSpace(req.ZipCode)

	if err := validateSetupUserReq(req); err != nil {
		return http.StatusBadRequest, err
	}

	uid, err := uuid.NewRandom()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to generate user id: %v", err)
	}
	userID := uid.String()
	user := &User{
		UserID:       userID,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		ZipCode:      req.ZipCode,
		TimestampSec: time.Now().Unix(),
	}

	if err := createUserInStorage(ctx, user); err != nil {
		return http.StatusInternalServerError, err
	}

	resp := &SetupUserResp{
		UserID: userID,
	}

	if err := EncodeResp(w, &resp); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

// GetUserInStorage checks that a userID exists in storage.
// Returns a non-nil error if storage client experienced a failure.
// If no error, returns true/false to indicate that userID exists or not.
func GetUserInStorage(ctx context.Context, userID string) (*User, bool, error) {
	client, err := StorageClient(ctx)
	if err != nil {
		return nil, false, err
	}
	defer client.Close()

	key := datastore.NameKey(UserKind, userID, nil)
	u := &User{}
	err = client.Get(ctx, key, &u)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			return nil, false, nil // userID does not exist
		}
		return nil, false, err
	}
	return u, true, nil // userID does exist
}

func validateSetupUserReq(req SetupUserReq) error {
	if req.FirstName == "" {
		return fmt.Errorf("missing first name")
	}
	if req.LastName == "" {
		return fmt.Errorf("missing last name")
	}
	if req.ZipCode == "" {
		return fmt.Errorf("missing zip code")
	}
	return validateZipCode(req.ZipCode)
}

func validateZipCode(zipCode string) error {
	return nil
}

func createUserInStorage(ctx context.Context, u *User) error {
	client, err := StorageClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	key := datastore.NameKey(UserKind, u.UserID, nil)
	_, err = client.Put(ctx, key, u)
	if err != nil {
		return fmt.Errorf("failed to create user in storage: %v", err)
	}
	return nil
}

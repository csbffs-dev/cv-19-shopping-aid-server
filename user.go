package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/google/uuid"
)

// User represents the user entity in storage.
// It stores the userID (key), first and last name, zipcode, and creation timestamp in seconds.
type User struct {
	UserID       string `datastore:"userID" json:"user_id"`
	FirstName    string `datastore:"firstName" json:"first_name"`
	LastName     string `datastore:"lastName" json:"last_name"`
	ZipCode      string `datastore:"zipCode" json:"zip_code"`
	TimestampSec int64  `datastore:"timestampSec" json:"timestamp_sec"`
}

// ******************************************
// ** BEGIN SetupUser
// ******************************************

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

// SetupUser sets up a user in storage.
func SetupUser(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	var req SetupUserReq
	if err := DecodeReq(r.Body, &req); err != nil {
		return http.StatusBadRequest, err
	}

	if err := validateSetupUserReq(&req); err != nil {
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

	if err := createOrUpdateUserInStorage(ctx, user); err != nil {
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

func validateSetupUserReq(req *SetupUserReq) error {
	req.FirstName = strings.TrimSpace(req.FirstName)
	if req.FirstName == "" {
		return fmt.Errorf("missing first name")
	}
	req.LastName = strings.TrimSpace(req.LastName)
	if req.LastName == "" {
		return fmt.Errorf("missing last name")
	}
	req.ZipCode = strings.TrimSpace(req.ZipCode)
	if req.ZipCode == "" {
		return fmt.Errorf("missing zip code")
	}
	return validateZipCode(req.ZipCode)
}

func validateZipCode(zipCode string) error {
	s := "zip code does not follow basic format"
	if len(zipCode) != 5 {
		return fmt.Errorf("%s: must contain 5 digits", s)
	}
	if _, err := strconv.Atoi(zipCode); err != nil {
		return fmt.Errorf("%s: %v", s, err)
	}
	return nil
}

// ******************************************
// ** END SetupUser
// ******************************************

// ******************************************
// ** BEGIN EditUser
// ******************************************

type EditUserReq struct {
	UserID    string `json:"user_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	ZipCode   string `json:"zip_code"`
}

func EditUser(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	var req EditUserReq
	if err := DecodeReq(r.Body, &req); err != nil {
		return http.StatusBadRequest, err
	}
	if err := validateEditUserReq(&req); err != nil {
		return http.StatusBadRequest, err
	}

	u, ok, err := GetUserInStorage(ctx, req.UserID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to query storage: %v", err)
	}
	if !ok {
		return http.StatusForbidden, fmt.Errorf("user id is invalid: %q", req.UserID)
	}

	u.FirstName = req.FirstName
	u.LastName = req.LastName
	u.ZipCode = req.ZipCode

	if err := createOrUpdateUserInStorage(ctx, u); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func validateEditUserReq(req *EditUserReq) error {
	if req.UserID == "" {
		return fmt.Errorf("missing user id")
	}
	req.FirstName = strings.TrimSpace(req.FirstName)
	if req.FirstName == "" {
		return fmt.Errorf("missing first name")
	}
	req.LastName = strings.TrimSpace(req.LastName)
	if req.LastName == "" {
		return fmt.Errorf("missing last name")
	}
	req.ZipCode = strings.TrimSpace(req.ZipCode)
	if req.ZipCode == "" {
		return fmt.Errorf("missing zip code")
	}
	return nil
}

// ******************************************
// ** END EditUser
// ******************************************

// ******************************************
// ** BEGIN DeleteUser
// ******************************************

type DeleteUserReq struct {
	UserID string `json:"user_id"`
}

func DeleteUser(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	var req DeleteUserReq
	if err := DecodeReq(r.Body, &req); err != nil {
		return http.StatusBadRequest, err
	}
	if err := validateDeleteUserReq(&req); err != nil {
		return http.StatusBadRequest, err
	}

	_, ok, err := GetUserInStorage(ctx, req.UserID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to query storage: %v", err)
	}
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("user id is invalid: %q", req.UserID)
	}

	if err := deleteUserInStorage(ctx, req.UserID); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func validateDeleteUserReq(req *DeleteUserReq) error {
	if req.UserID == "" {
		return fmt.Errorf("missing user id")
	}
	return nil
}

// ******************************************
// ** END DeleteUser
// ******************************************

// ******************************************
// ** BEGIN QueryUser
// ******************************************

type QueryUserReq struct {
	UserID string `json:"user_id"`
}

type QueryUserResp struct {
	UserInfo *User `json:"user"`
}

func QueryUser(ctx context.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	var req QueryUserReq
	if err := DecodeReq(r.Body, &req); err != nil {
		return http.StatusBadRequest, err
	}
	if err := validateQueryUserReq(&req); err != nil {
		return http.StatusBadRequest, err
	}
	u, ok, err := GetUserInStorage(ctx, req.UserID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to query storage: %v", err)
	}
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("user id is invalid: %q", req.UserID)
	}
	if err := EncodeResp(w, &QueryUserResp{UserInfo: u}); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func validateQueryUserReq(req *QueryUserReq) error {
	if req.UserID == "" {
		return fmt.Errorf("missing user id")
	}
	return nil
}

// ******************************************
// ** END QueryUser
// ******************************************

// GetUserInStorage fetches the user in with key = userID in storage.
// Returns a non-nil error if storage client experienced a failure.
// If no error, returns true/false to indicate that userID exists or not.
func GetUserInStorage(ctx context.Context, userID string) (*User, bool, error) {
	client, err := StorageClient(ctx)
	if err != nil {
		return nil, false, err
	}
	defer client.Close()

	key := datastore.NameKey(UserKind, userID, nil)
	var u User
	err = client.Get(ctx, key, &u)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			return nil, false, nil // userID does not exist
		}
		return nil, false, err // storage error
	}
	return &u, true, nil // userID does exist
}

// createOrUpdateUserInStorage puts the user with key = userID in storage.
func createOrUpdateUserInStorage(ctx context.Context, u *User) error {
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

// deleteUserInStorage deletes the user with key = userID in storage.
func deleteUserInStorage(ctx context.Context, userID string) error {
	client, err := StorageClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	key := datastore.NameKey(UserKind, userID, nil)
	if err := client.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete user in storage: %v", err)
	}
	return nil
}

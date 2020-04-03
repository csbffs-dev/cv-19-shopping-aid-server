package main

import (
	"context"
	"log"
	"net/http"
	"os"
)

func main() {
	// TODO: Use github.com/gorilla/mux for HTTP routing.
	http.HandleFunc("/user/setup", userSetupHandler)
	http.HandleFunc("/user/edit", userEditHandler)
	http.HandleFunc("/user/delete", userDeleteHandler)
	http.HandleFunc("/user/query", userQueryHandler)
	http.HandleFunc("/item/query", itemQueryHandler)
	http.HandleFunc("/store/query", storeQueryHandler)
	http.HandleFunc("/store/add", storeAddHandler)
	http.HandleFunc("/report/upload", reportUploadHandler)
	http.HandleFunc("/receipt/parse", receiptParseHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func userSetupHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	if status, err := SetupUser(ctx, w, r); err != nil {
		http.Error(w, err.Error(), status)
	}
}

func userEditHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	if status, err := EditUser(ctx, w, r); err != nil {
		http.Error(w, err.Error(), status)
	}
}

func userDeleteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	if status, err := DeleteUser(ctx, w, r); err != nil {
		http.Error(w, err.Error(), status)
	}
}


func userQueryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	if status, err := QueryUser(ctx, w, r); err != nil {
		http.Error(w, err.Error(), status)
	}
}

func itemQueryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	status, err := QueryItems(ctx, w, r)
	if err != nil {
		http.Error(w, err.Error(), status)
	}
}

func storeQueryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	status, err := QueryStores(ctx, w, r)
	if err != nil {
		http.Error(w, err.Error(), status)
	}
}

func storeAddHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	status, err := AddStore(ctx, w, r)
	if err != nil {
		http.Error(w, err.Error(), status)
	}
}

func reportUploadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	status, err := UploadReport(ctx, w, r)
	if err != nil {
		http.Error(w, err.Error(), status)
	}
}

func receiptParseHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	status, err := ParseReceipt(ctx, w, r)
	if err != nil {
		http.Error(w, err.Error(), status)
	}
}

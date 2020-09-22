package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
)

var db *bolt.DB

type shortLink struct {
	ID          string `json:"id"`
	Destination string `json:"destination"`
}

func createURL(w http.ResponseWriter, r *http.Request) {
	var desiredLink shortLink
	err := json.NewDecoder(r.Body).Decode(&desiredLink)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, err = rand.Read(make([]byte, 4))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	desiredLink.ID = base64.URLEncoding.EncodeToString(randomBytes)
	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("TUS"))
		if err != nil {
			return err
		}
		err = bucket.Put([]byte(desiredLink.ID), []byte(desiredLink.Destination))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
	jsonResponse, _ := json.Marshal(desiredLink)
	w.Write(jsonResponse)
}

func redirectToURL(w http.ResponseWriter, r *http.Request) {
	pathValues := mux.Vars(r)
	var destination []byte
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("TUS"))
		if bucket == nil {
			return errors.New("Bucket not found")
		}
		o := bucket.Get([]byte(pathValues["ID"]))
		destination = append(destination, o...)
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if string(destination) == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	http.Redirect(w, r, string(destination), http.StatusMovedPermanently)
}

func main() {
	var err error
	db, err = bolt.Open("urlshortner.db", 0644, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	r := mux.NewRouter()
	r.HandleFunc("/", createURL).Methods("POST")
	r.HandleFunc("/{ID}", redirectToURL).Methods("GET")
	server := http.Server{
		Addr:         ":8080",
		Handler:      r,
		IdleTimeout:  30 * time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}
	err = server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

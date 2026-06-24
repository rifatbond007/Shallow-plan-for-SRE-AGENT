package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/user", getUserHandler)
	http.HandleFunc("/search", searchHandler)
	http.HandleFunc("/checkout", checkoutHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	user := fetchUser(r.URL.Query().Get("id"))
	json.NewEncoder(w).Encode(user)
}

func fetchUser(id string) *User {
	if id == "" {
		return nil
	}
	return &User{ID: 1, Name: "Alice"}
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	results, err := searchDB(q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(results)
}

func searchDB(q string) ([]string, error) {
	query := fmt.Sprintf("SELECT name FROM items WHERE name LIKE '%%%s%%'", q)
	db, err := sql.Open("postgres", "host=localhost user=app password=secret dbname=shop sslmode=disable")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		results = append(results, name)
	}
	return results, nil
}

func checkoutHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := http.Get("http://payment-service/charge")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	fmt.Fprintf(w, "checkout ok")
	_ = time.Now()
}

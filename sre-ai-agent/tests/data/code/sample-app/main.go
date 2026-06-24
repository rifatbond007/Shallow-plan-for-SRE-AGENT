package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	http.HandleFunc("/user", getUserHandler)
	http.HandleFunc("/search", searchHandler)
	http.HandleFunc("/checkout", checkoutHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/order", orderHandler)
	http.HandleFunc("/review", reviewHandler)
	http.HandleFunc("/profile", profileHandler)
	http.HandleFunc("/notify", notifyHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// BUG 1: getUserHandler doesn't check nil from fetchUser
func getUserHandler(w http.ResponseWriter, r *http.Request) {
	user := fetchUser(r.URL.Query().Get("id"))
	json.NewEncoder(w).Encode(user)
}

// BUG 2: fetchUser returns nil on empty id — caller will crash
func fetchUser(id string) *User {
	if id == "" {
		return nil
	}
	return &User{ID: 1, Name: "Alice"}
}

// BUG 3: searchDB has SQL injection via fmt.Sprintf
// BUG 4: searchDB doesn't check rows.Err() after loop
// BUG 5: searchDB opens new connection every call (no pool reuse)
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
	// BUG 4: missing rows.Err() check — silent data loss
	return results, nil
}

// BUG 6: checkoutHandler uses default HTTP client (no timeout)
// BUG 7: no context timeout — can hang forever on slow upstream
func checkoutHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := http.Get("http://payment-service/charge")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	fmt.Fprintf(w, "checkout ok")
}

// BUG 8: healthHandler can crash from panic — no recovery middleware
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("deep") == "true" {
		checkDBHealth()
	}
	fmt.Fprintf(w, "ok")
}

func checkDBHealth() {
	var db *sql.DB
	db.Ping() // nil pointer dereference — crash
}

// BUG 9: orderHandler spawns goroutines without tracking (leak on retry)
func orderHandler(w http.ResponseWriter, r *http.Request) {
	orderID := r.URL.Query().Get("id")
	go processOrder(orderID)
	fmt.Fprintf(w, "order submitted")
}

func processOrder(id string) {
	time.Sleep(5 * time.Second)
	log.Printf("order %s processed", id)
}

// BUG 10: reviewHandler uses user input in file path (path traversal)
func reviewHandler(w http.ResponseWriter, r *http.Request) {
	productID := r.URL.Query().Get("product_id")
	data, err := os.ReadFile("/data/reviews/" + productID + ".json")
	if err != nil {
		http.Error(w, "review not found", http.StatusNotFound)
		return
	}
	w.Write(data)
}

// BUG 11: profileHandler doesn't close HTTP response body (connection leak)
func profileHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("id")
	resp, err := http.Get("http://profile-service/user?id=" + userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// BUG: resp.Body never closed — connection leak
	io.Copy(w, resp.Body)
}

// BUG 12: notifyHandler has no context — can hang on slow notification service
func notifyHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := http.Post("http://notification-service/send", "application/json", r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	fmt.Fprintf(w, "notified")
}


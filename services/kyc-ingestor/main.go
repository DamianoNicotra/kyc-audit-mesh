package main

import (
    "encoding/json"
    "log"
    "net/http"
    "time"

    "github.com/gorilla/mux"
)

type KYCEvent struct {
    ID        string    `json:"id"`
    UserID    string    `json:"user_id"`
    Action    string    `json:"action"`
    Timestamp time.Time `json:"timestamp"`
}

var events []KYCEvent

func postKYCEvent(w http.ResponseWriter, r *http.Request) {
    var event KYCEvent
    if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }
    event.ID = time.Now().Format("20060102150405.000")
    event.Timestamp = time.Now()
    events = append(events, event)
    
    log.Printf("Event received: user=%s action=%s", event.UserID, event.Action)
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "accepted",
        "id":      event.ID,
    })
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "status":  "healthy",
        "service": "kyc-ingestor",
    })
}

func main() {
    log.Println("KYC Ingestor starting...")
    
    r := mux.NewRouter()
    r.HandleFunc("/health", healthHandler).Methods("GET")
    r.HandleFunc("/kyc/event", postKYCEvent).Methods("POST")
    
    log.Println("Server listening on :8080")
    if err := http.ListenAndServe(":8080", r); err != nil {
        log.Fatal(err)
    }
}

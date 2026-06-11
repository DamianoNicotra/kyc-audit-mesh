
package main

import (
    "crypto/sha256"
    "database/sql"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"

    "github.com/gorilla/mux"
    _ "github.com/snowflakedb/gosnowflake"
)

type KYCEvent struct {
    ID        string                 `json:"id"`
    UserID    string                 `json:"user_id"`
    Action    string                 `json:"action"`
    Details   map[string]interface{} `json:"details"`
    Timestamp time.Time              `json:"timestamp"`
    IPAddress string                 `json:"ip_address"`
}

var db *sql.DB
var eventsStore []KYCEvent

func initDB() error {
    dsn := fmt.Sprintf(
        "%s:%s@%s/%s?schema=%s&warehouse=%s&role=%s",
        "terraform_user",
        "TerraformSnowflake2026!",
        "hjdlbku-cv00477.snowflakecomputing.com",
        "SPRING_SNOWFLAKE_DB",
        "PUBLIC",
        "TINY_WAREHOUSE",
        "TERRAFORM_ROLE",
    )
    var err error
    db, err = sql.Open("snowflake", dsn)
    if err != nil {
        return err
    }
    return db.Ping()
}

func getLastHashFromSnowflake(userID string) (string, error) {
    var lastHash sql.NullString
    err := db.QueryRow(`
        SELECT HASH FROM SPRING_SNOWFLAKE_DB.PUBLIC.KYC_AUDIT
        WHERE USER_ID = ?
        ORDER BY TIMESTAMP DESC
        LIMIT 1
    `, userID).Scan(&lastHash)
    if err == sql.ErrNoRows {
        return "", nil
    }
    if err != nil {
        return "", err
    }
    return lastHash.String, nil
}

func saveToSnowflake(event KYCEvent, prevHash, currentHash string) {
    detailsJSON, _ := json.Marshal(event.Details)
    
    stmt, err := db.Prepare(`
        INSERT INTO SPRING_SNOWFLAKE_DB.PUBLIC.KYC_AUDIT 
        (USER_ID, ACTION, DETAILS, PREVIOUS_HASH, HASH, TIMESTAMP)
        VALUES (?, ?, ?, ?, ?, ?)
    `)
    if err != nil {
        log.Printf("Snowflake prepare error: %v", err)
        return
    }
    defer stmt.Close()
    
    _, err = stmt.Exec(event.UserID, event.Action, string(detailsJSON), prevHash, currentHash, event.Timestamp)
    if err != nil {
        log.Printf("Snowflake error (non-blocking): %v", err)
    } else {
        log.Printf("Saved to Snowflake: user=%s action=%s", event.UserID, event.Action)
    }
}

func computeHash(prevHash, userID, action, details, timestamp string) string {
    data := prevHash + userID + action + details + timestamp
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:])
}

func postKYCEvent(w http.ResponseWriter, r *http.Request) {
    var event KYCEvent
    if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    event.ID = fmt.Sprintf("evt-%d", time.Now().UnixNano())
    event.Timestamp = time.Now()

    prevHash := ""
    if len(eventsStore) > 0 {
        detailsJSON, _ := json.Marshal(eventsStore[len(eventsStore)-1].Details)
        prevHash = computeHash("", eventsStore[len(eventsStore)-1].UserID, 
            eventsStore[len(eventsStore)-1].Action, string(detailsJSON), 
            eventsStore[len(eventsStore)-1].Timestamp.Format(time.RFC3339Nano))
    }
    
    detailsJSON, _ := json.Marshal(event.Details)
    timestampStr := event.Timestamp.Format(time.RFC3339Nano)
    currentHash := computeHash(prevHash, event.UserID, event.Action, string(detailsJSON), timestampStr)

    eventsStore = append(eventsStore, event)

    // Salva in Snowflake in modo non bloccante
    go saveToSnowflake(event, prevHash, currentHash)

    log.Printf("Event stored: user=%s action=%s hash=%s prev_hash=%s",
        event.UserID, event.Action, currentHash[:8],
        func() string { if prevHash != "" { return prevHash[:8] } else { return "none" } }())

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "status":        "accepted",
        "id":            event.ID,
        "hash":          currentHash[:16] + "...",
        "previous_hash": func() string { if prevHash != "" { return prevHash[:16] + "..." } else { return "none" } }(),
    })
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "status":  "healthy",
        "service": "kyc-ingestor",
    })
}

func eventsHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(eventsStore)
}

func main() {
    if err := initDB(); err != nil {
        log.Printf("Warning: Snowflake connection failed: %v", err)
    } else {
        log.Println("Connected to Snowflake")
        defer db.Close()
    }

    r := mux.NewRouter()
    r.HandleFunc("/health", healthHandler).Methods("GET")
    r.HandleFunc("/kyc/event", postKYCEvent).Methods("POST")
    r.HandleFunc("/kyc/events", eventsHandler).Methods("GET")

    log.Println("KYC Ingestor starting on :8080")
    if err := http.ListenAndServe(":8080", r); err != nil {
        log.Fatal(err)
    }
}


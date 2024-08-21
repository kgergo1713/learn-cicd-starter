package main

import (
 "database/sql"
 "fmt"
 "log"
 "net/url"
 "os"

 _ "github.com/lib/pq" // Import the PostgreSQL driver
)

type apiConfig struct {
 DB *sql.DB
}

func main() {
 // Step 1: Read the DATABASE_URL from the environment variable
 databaseURL := os.Getenv("DATABASE_URL")
 if databaseURL == "" {
  log.Fatal("DATABASE_URL is not set")
 }

 // Step 2: Parse the URL to extract necessary components
 parsedURL, err := url.Parse(databaseURL)
 if err != nil {
  log.Fatalf("Failed to parse DATABASE_URL: %v", err)
 }

 // Step 3: Use the parsed information to establish a connection to the database
 db, err := sql.Open("postgres", parsedURL.String())
 if err != nil {
  log.Fatalf("Failed to connect to the database: %v", err)
 }
 defer db.Close()

 // Verify the connection
 err = db.Ping()
 if err != nil {
  log.Fatalf("Failed to ping the database: %v", err)
 }

 fmt.Println("Successfully connected to the database!")
}
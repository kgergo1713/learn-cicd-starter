package main

import (
	"bufio"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/bootdotdev/learn-cicd-starter/internal/database"

	_ "github.com/lib/pq" // Import the PostgreSQL driver
)

type apiConfig struct {
	DB *database.Queries
}

//go:embed static/*
var staticFiles embed.FS

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp   time.Time    `json:"timestamp"`
	TextPayload string       `json:"textPayload"`
	Severity    string       `json:"severity,omitempty"`
	HTTPRequest *HTTPRequest `json:"httpRequest,omitempty"`
}

// HTTPRequest represents HTTP request details in a log entry
type HTTPRequest struct {
	RequestMethod string `json:"requestMethod"`
	RequestURL    string `json:"requestUrl"`
	Status        int    `json:"status"`
	UserAgent     string `json:"userAgent"`
	RemoteIP      string `json:"remoteIp"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // default to 8080 if PORT is not set
	}

	apiCfg := apiConfig{}

	dbURL := os.Getenv("DATABASE_URL")
	fmt.Println("DATABASE_URL:", dbURL)
	if dbURL == "" {
		log.Println("DATABASE_URL environment variable is not set")
		log.Println("Running without CRUD endpoints")
	} else {
		parsedURL, err := addParseTimeParam(dbURL)
		if err != nil {
			log.Fatal(err)
		}
		db, err := sql.Open("postgres", parsedURL)
		if err != nil {
			log.Fatal(err)
		}
		err = db.Ping()
		if err != nil {
			log.Fatalf("Could not connect to the database: %v", err)
		}
		dbQueries := database.New(db)
		apiCfg.DB = dbQueries
		log.Println("Connected to database!")
	}

	router := chi.NewRouter()

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		f, err := staticFiles.Open("static/index.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer f.Close()
		if _, err := io.Copy(w, f); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	v1Router := chi.NewRouter()

	if apiCfg.DB != nil {
		v1Router.Post("/users", apiCfg.handlerUsersCreate)
		v1Router.Get("/users", apiCfg.middlewareAuth(apiCfg.handlerUsersGet))
		v1Router.Get("/notes", apiCfg.middlewareAuth(apiCfg.handlerNotesGet))
		v1Router.Post("/notes", apiCfg.middlewareAuth(apiCfg.handlerNotesCreate))
	}

	v1Router.Get("/healthz", handlerReadiness)

	router.Mount("/v1", v1Router)
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: time.Second * 5, // use seconds or it will default to nanoseconds
	}

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(srv.ListenAndServe())
}

func addParseTimeParam(input string) (string, error) {
	const dummyScheme = "http://"
	if !strings.Contains(input, dummyScheme) {
		input = "http://" + input
	}
	u, err := url.Parse(input)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Add("parseTime", "true")
	u.RawQuery = q.Encode()
	returnUrl := u.String()
	returnUrl = strings.TrimPrefix(returnUrl, dummyScheme)
	return returnUrl, nil
}

func parseLogs(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Connected to database!") || strings.Contains(line, "Serving on port:") {
			fmt.Println("Info:", line)
		} else if strings.Contains(line, "POST") || strings.Contains(line, "ERROR") {
			var logEntry LogEntry
			err := json.Unmarshal([]byte(line), &logEntry)
			if err != nil {
				fmt.Println("Error parsing log entry:", err)
				continue
			}
			fmt.Printf("Parsed Log Entry: %+v\n", logEntry)
			if strings.Contains(logEntry.TextPayload, "connection refused") {
				handleConnectionRefused(logEntry)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}
}

func handleConnectionRefused(logEntry LogEntry) {
	fmt.Println("Handling connection refused error:", logEntry.TextPayload)
	// Add your error handling logic here
}
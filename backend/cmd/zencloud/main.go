package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

const (
	uploadDir = "./uploads"
)

var db *sql.DB

// FileMeta represents file metadata stored in the database.
type FileMeta struct {
	ID         string    `json:"id"`
	FileName   string    `json:"filename"`
	Extension  string    `json:"extension"`
	UploadDate time.Time `json:"upload_date"`
}

func initDB() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v\n", err)
	}

	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	if dbUser == "" || dbPass == "" {
		log.Println("DB_USER or DB_PASS is not set in the environment")
		return
	}

	dsn := fmt.Sprintf("postgres://%s:%s@db:5432/db?sslmode=disable", dbUser, dbPass)
	// Assign to the package-level variable (remove the ':=')
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	// ensure table exists
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS filemeta (
            id         TEXT PRIMARY KEY,
            filename   TEXT,
            extension  TEXT,
            upload_date TIMESTAMP
        );
    `)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	err := r.ParseMultipartForm(10 << 26) // 640 MB max
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File not provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// generate unique file id
	id := uuid.New().String()
	ext := filepath.Ext(handler.Filename)

	// final file path: save with uuid to avoid collisions
	filePath := filepath.Join(uploadDir, id+ext)
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// copy file content
	if _, err = io.Copy(dst, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// save metadata to the database
	meta := FileMeta{
		ID:         id,
		FileName:   handler.Filename,
		Extension:  ext,
		UploadDate: time.Now(),
	}
	_, err = db.Exec(`
        INSERT INTO filemeta (id, filename, extension, upload_date)
        VALUES ($1, $2, $3, $4)`,
		meta.ID, meta.FileName, meta.Extension, meta.UploadDate)
	if err != nil {
		http.Error(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meta)
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve the id parameter
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	// Lookup metadata in db
	var fileMeta FileMeta
	row := db.QueryRow(`SELECT id, filename, extension, upload_date FROM filemeta WHERE id=$1`, id)
	if err := row.Scan(&fileMeta.ID, &fileMeta.FileName, &fileMeta.Extension,
		&fileMeta.UploadDate); err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	filePath := filepath.Join(uploadDir, fileMeta.ID+fileMeta.Extension)
	http.ServeFile(w, r, filePath)
}

func listFilesHandler(w http.ResponseWriter, r *http.Request) {
	// CORS headers already added
	rows, err := db.Query(`SELECT id, filename, extension, upload_date FROM filemeta`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var files []FileMeta
	for rows.Next() {
		var meta FileMeta
		if err := rows.Scan(&meta.ID, &meta.FileName, &meta.Extension, &meta.UploadDate); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		files = append(files, meta)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func deleteFileHandler(w http.ResponseWriter, r *http.Request) {
	// Expecting DELETE method
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// We expect an id query parameter, e.g., /delete?id={fileID}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	// get the file's metadata so we can remove the file from disk
	var fileMeta FileMeta
	row := db.QueryRow(`SELECT id, filename, extension, upload_date FROM filemeta WHERE id=$1`, id)
	if err := row.Scan(&fileMeta.ID, &fileMeta.FileName, &fileMeta.Extension, &fileMeta.UploadDate); err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Delete file from disk
	filePath := filepath.Join(uploadDir, fileMeta.ID+fileMeta.Extension)
	if err := os.Remove(filePath); err != nil {
		http.Error(w, "Error deleting file from disk: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Remove metadata from the database
	_, err := db.Exec(`DELETE FROM filemeta WHERE id=$1`, id)
	if err != nil {
		http.Error(w, "Error deleting metadata: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "File deleted successfully")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		// Handle pre-flight request
		if r.Method == http.MethodOptions {
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	// ensure upload directory exists
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.Mkdir(uploadDir, os.ModePerm)
	}
	initDB()
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/upload", uploadHandler)
	mux.HandleFunc("/download", downloadHandler)
	mux.HandleFunc("/files", listFilesHandler)
	mux.HandleFunc("/delete", deleteFileHandler) 

	handler := corsMiddleware(mux)

	fmt.Println("Server running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

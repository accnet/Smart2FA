package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"smart2fa/internal/db"
	"smart2fa/internal/handlers"
)

type tmpl struct {
	t *template.Template
}

func (t *tmpl) Render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.t.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template %s: %v", name, err)
		http.Error(w, "template error", 500)
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := filepath.Join("data", "smart2fa.db")
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatal(err)
	}
	db.Init(dbPath)

	// Parse all templates
	t := template.Must(template.ParseGlob("templates/*.html"))
	tr := &tmpl{t: t}

	mux := http.NewServeMux()

	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Routes
	mux.HandleFunc("GET /{$}", handlers.GetLock(tr))
	mux.HandleFunc("POST /unlock", handlers.PostUnlock(tr))
	mux.HandleFunc("GET /dashboard", handlers.GetDashboard(tr))
	mux.HandleFunc("GET /partial/codes", handlers.GetCodes())
	mux.HandleFunc("POST /entry/add", handlers.PostAddEntry(tr))
	mux.HandleFunc("POST /entry/edit", handlers.PostEditEntry())
	mux.HandleFunc("POST /entry/delete", handlers.PostDeleteEntry())
	mux.HandleFunc("POST /group/rename", handlers.PostRenameGroup())
	mux.HandleFunc("POST /lock", handlers.PostLock())
	mux.HandleFunc("GET /backup", handlers.GetBackupPage(tr))
	mux.HandleFunc("POST /backup/export", handlers.PostExport())
	mux.HandleFunc("POST /backup/import", handlers.PostImport(tr))

	log.Printf("Smart2FA running on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

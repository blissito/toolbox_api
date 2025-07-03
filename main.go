package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"toolbox/api"
	"toolbox/database"

	_ "modernc.org/sqlite"
)

// DB es la conexión global a la base de datos
var DB *sql.DB

// Configuración
type Config struct {
	JWTSecret string
}

// dashboardHandler maneja las solicitudes al dashboard
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Configurar encabezados de seguridad
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

	// Si es una solicitud de archivo estático, servirlo directamente
	if strings.HasPrefix(r.URL.Path, "/dashboard/static/") {
		http.StripPrefix("/dashboard/static/", http.FileServer(http.Dir("static/dashboard"))).ServeHTTP(w, r)
		return
	}

	// Para cualquier otra ruta bajo /dashboard, servir el index.html
	http.ServeFile(w, r, "static/dashboard/index.html")
}

// homeHandler maneja la página de inicio
func homeHandler(w http.ResponseWriter, r *http.Request) {
	// Solo manejar la ruta raíz
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Configurar encabezados de seguridad
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

	// Servir el archivo home.html
	http.ServeFile(w, r, "home.html")
}

// docsHandler maneja las rutas de documentación
func docsHandler(w http.ResponseWriter, r *http.Request) {
	// Configurar encabezados de seguridad
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

	// Extraer la ruta solicitada
	path := r.URL.Path

	// Servir el archivo webfetch.html para cualquier ruta que comience con /docs/
	if strings.HasPrefix(path, "/docs/") {
		// Si es la raíz de docs, redirigir a webfetch
		if path == "/docs/" || path == "/docs" {
			http.ServeFile(w, r, "docs/webfetch.html")
			return
		}

		// Intentar servir archivos estáticos desde la carpeta docs
		filePath := filepath.Join("docs", strings.TrimPrefix(path, "/docs/"))
		if _, err := os.Stat(filePath); err == nil {
			http.ServeFile(w, r, filePath)
			return
		}

		// Si no se encuentra el archivo, servir webfetch.html
		http.ServeFile(w, r, "docs/webfetch.html")
		return
	}

	// Si no es una ruta de documentación, devolver 404
	http.NotFound(w, r)
}

func main() {
	// Crear directorio de datos si no existe
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatalf("Error al crear directorio de datos: %v", err)
	}

	// Configurar ruta de la base de datos
	var dbPath string
	if os.Getenv("FLY") == "true" {
		// En producción (Fly.io), usar ruta del volumen
		dbPath = "/data/toolbox.db"
	} else {
		// En desarrollo, usar ruta relativa
		dbPath = "data/toolbox.db"
	}
	dbPath += "?cache=shared&_journal=WAL&_busy_timeout=5000&_foreign_keys=on"

	// Inicializar base de datos
	DB, err := database.Init(dbPath)
	if err != nil {
		log.Fatalf("Error al inicializar la base de datos: %v", err)
	}
	defer database.Close(DB)

	// Ejecutar migraciones
	if err := database.RunMigrations(DB); err != nil {
		log.Fatalf("Error al ejecutar migraciones: %v", err)
	}

	// Middleware CORS
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

			// Manejar solicitudes preflight
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	// Crear un nuevo enrutador
	mux := http.NewServeMux()

	// Configurar rutas de la API
	api.SetupRoutes(mux, DB)

	// Ruta de health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Verificar conexión a la base de datos
		if err := DB.Ping(); err != nil {
			http.Error(w, "Database connection error", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Configurar rutas principales
	mux.HandleFunc("/", homeHandler)

	// Redireccionar /dash a /dashboard
	mux.HandleFunc("/dash", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/dashboard", dashboardHandler)
	mux.HandleFunc("/dashboard/", dashboardHandler)

	// Configurar ruta de documentación
	mux.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/webfetch", http.StatusMovedPermanently)
	})

	// Ruta de login
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/login.html")
	})

	// Servir archivos estáticos
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Servir archivos de documentación
	mux.Handle("/docs/", http.StripPrefix("/docs/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Configurar encabezados de seguridad
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Si es la raíz de docs, redirigir a webfetch
		if r.URL.Path == "" || r.URL.Path == "/" {
			http.Redirect(w, r, "/docs/webfetch", http.StatusFound)
			return
		}

		// Si es webfetch, servir webfetch.html
		if r.URL.Path == "webfetch" || r.URL.Path == "webfetch/" {
			http.ServeFile(w, r, "docs/webfetch.html")
			return
		}

		// Si es duckduckgo_search, servir duckduckgo_search.html si existe
		if r.URL.Path == "duckduckgo_search" || r.URL.Path == "duckduckgo_search/" {
			http.ServeFile(w, r, "docs/duckduckgo_search.html")
			return
		}

		// Intentar servir el archivo estático solicitado
		filePath := filepath.Join("docs", r.URL.Path)
		if _, err := os.Stat(filePath); err == nil {
			http.ServeFile(w, r, filePath)
			return
		}

		// Si no se encuentra, servir webfetch.html
		http.ServeFile(w, r, "docs/webfetch.html")
	})))

	// Aplicar CORS a todas las rutas
	handler := corsMiddleware(mux)

	// Obtener el puerto de la variable de entorno o usar 8000 por defecto
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	// Iniciar el servidor con el manejador CORS
	log.Printf("Servidor iniciado en http://0.0.0.0:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

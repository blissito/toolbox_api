package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

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

	// Servir el archivo del dashboard
	http.ServeFile(w, r, "static/dash.html")
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

func docsHandler(w http.ResponseWriter, r *http.Request) {
	// Servir el archivo de documentación
	if r.URL.Path == "/docs/webfetch" || r.URL.Path == "/docs/webfetch/" {
		http.ServeFile(w, r, "docs/webfetch.html")
		return
	}
	// Redirigir si se accede a /docs sin especificar el archivo
	http.Redirect(w, r, "/docs/webfetch", http.StatusFound)
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

	// Rutas de documentación
	mux.HandleFunc("/docs/webfetch", docsHandler)
	mux.HandleFunc("/docs/webfetch/", docsHandler)

	// Redirigir /docs a /docs/webfetch
	mux.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/webfetch", http.StatusMovedPermanently)
	})

	// Rutas principales
	mux.HandleFunc("/", homeHandler) // Ruta raíz
	mux.HandleFunc("/dash", dashboardHandler)
	mux.HandleFunc("/dash/", dashboardHandler)
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		// Servir el archivo de login desde el directorio static
		http.ServeFile(w, r, "static/login.html")
	})

	// Servir archivos estáticos
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Manejar rutas de archivos estáticos sin redirección
	mux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("static/js"))))
	mux.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("static/css"))))
	mux.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("static/images"))))
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("static/assets"))))

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

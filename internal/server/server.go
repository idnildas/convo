package server

import (
	"fmt"
	"net/http"
	"database/sql"

	"github.com/go-chi/chi/v5"

	"convo/internal/middleware"
	"convo/internal/handlers"
	"convo/internal/handlers/auth"
	"convo/internal/handlers/user"
	"convo/internal/handlers/preprocess"
	"convo/internal/handlers/room"

)

type Server struct {
	Addr string
	DB  *sql.DB // Assuming you want to use a database connection
	JWTSecret string
	JWTTTLHrs int
}

func NewServer(addr string, db *sql.DB, jwtSecret string, jwtTTL int) *Server {
	return &Server{
		Addr: addr,
		DB:   db,
		JWTSecret: jwtSecret,
		JWTTTLHrs: jwtTTL,
	}
}

func HandlerFunc(h http.Handler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        h.ServeHTTP(w, r)
    }
}

func (s *Server) Run() error {
	r := chi.NewRouter()

	// middlewares
	r.Use(middleware.Logger)

	// Mount routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintln(w, "Welcome to convo API! Server is running....")
	})
	r.Get("/health", handlers.HealthCheck)


	// auth routes (public)
	r.Route("/auth", func(r chi.Router) {
		r.Post("/signup", HandlerFunc(&auth.SignupHandler{DB: s.DB}))
		r.Post("/login", HandlerFunc(&auth.LoginHandler{
			DB:        s.DB,
			JWTSecret: s.JWTSecret,
			JWTTTLHrs: s.JWTTTLHrs,
		}))
	})

	// authenticated routes grouped by feature
	r.Route("/user", func(r chi.Router) {
		r.Use(middleware.AuthJWT(s.JWTSecret))
		r.Get("/me", HandlerFunc(&user.MeHandler{DB: s.DB}))
	})

	r.Route("/metadata", func(r chi.Router) {
		r.Use(middleware.AuthJWT(s.JWTSecret))
		r.Post("/", HandlerFunc(&preprocess.MetadataHandler{}))
	})

	r.Route("/rooms", func(r chi.Router) {
		r.Use(middleware.AuthJWT(s.JWTSecret))
		r.Post("/add", HandlerFunc(&room.CreateRoomHandler{DB: s.DB}))
		r.Post("/{id}/members", HandlerFunc(&room.AddMembersHandler{DB: s.DB}))
        r.Post("/{id}/send-message", HandlerFunc(&room.SendMessageHandler{DB: s.DB}))
        r.Get("/{id}/check", HandlerFunc(&room.RoomCheckHandler{DB: s.DB}))
		// future: r.Get("/", list rooms), r.Post("/{id}/join", join handler), etc.
		// future: r.Get("/", list rooms), r.Post("/{id}/join", join handler), etc.
	})

	// WebSocket endpoint (public)
	r.Get("/ws", handlers.Handler)

	fmt.Printf("Server running on %s\n", s.Addr)
	return http.ListenAndServe(s.Addr, r)
}



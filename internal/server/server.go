package server

import (
	"context"
	"net/http"

	"github.com/georgestander/ana-board/internal/board"
	"github.com/georgestander/ana-board/internal/store"
	"github.com/georgestander/ana-board/web"
)

const defaultBoardID = "default"

type Server struct {
	store  store.Store
	broker *Broker
}

func NewServer() (*Server, error) {
	return NewServerWithStore(store.NewMemoryStore())
}

func NewServerWithStore(st store.Store) (*Server, error) {
	frame, err := board.NewFrame(board.DefaultRows, board.DefaultCols)
	if err != nil {
		return nil, err
	}

	if err := st.SaveCurrentFrame(context.Background(), defaultBoardID, frame); err != nil {
		return nil, err
	}

	return &Server{
		store:  st,
		broker: NewBroker(),
	}, nil
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	staticFiles, err := web.StaticFiles()
	if err == nil {
		mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFiles))))
	}

	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /admin", s.handleAdmin)
	mux.HandleFunc("POST /admin/messages", s.handleAdminCreateMessage)
	mux.HandleFunc("POST /admin/clear", s.handleAdminClear)
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /api/current", s.handleCurrentFrame)
	mux.HandleFunc("POST /api/messages", s.handleCreateMessage)
	mux.HandleFunc("GET /api/messages", s.handleListMessages)
	mux.HandleFunc("POST /api/clear", s.handleAPIClear)
	mux.HandleFunc("GET /events", s.handleEvents)

	return mux
}

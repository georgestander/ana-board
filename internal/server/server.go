package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/georgestander/ana-board/internal/board"
	"github.com/georgestander/ana-board/internal/store"
	"github.com/georgestander/ana-board/web"
)

const defaultBoardID = "default"
const DefaultMaxSubscribers = 64

type Server struct {
	store          store.Store
	broker         *Broker
	trustedOrigins map[string]struct{}
}

type Option func(*Server) error

func WithTrustedOrigins(origins []string) Option {
	return func(s *Server) error {
		for _, raw := range origins {
			origin, err := normalizeTrustedOrigin(raw)
			if err != nil {
				return err
			}
			if origin == "" {
				continue
			}

			s.trustedOrigins[origin] = struct{}{}
		}

		return nil
	}
}

func WithMaxSubscribers(maxSubscribers int) Option {
	return func(s *Server) error {
		if maxSubscribers <= 0 {
			return fmt.Errorf("max subscribers must be positive")
		}

		s.broker = NewBrokerWithLimit(maxSubscribers)
		return nil
	}
}

func NewServer(opts ...Option) (*Server, error) {
	return NewServerWithStore(store.NewMemoryStore(), opts...)
}

func NewServerWithStore(st store.Store, opts ...Option) (*Server, error) {
	frame, err := board.NewFrame(board.DefaultRows, board.DefaultCols)
	if err != nil {
		return nil, err
	}

	if err := st.SaveCurrentFrame(context.Background(), defaultBoardID, frame); err != nil {
		return nil, err
	}

	srv := &Server{
		store:          st,
		broker:         NewBroker(),
		trustedOrigins: make(map[string]struct{}),
	}

	for _, opt := range opts {
		if err := opt(srv); err != nil {
			return nil, err
		}
	}

	return srv, nil
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

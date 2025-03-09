package p2p

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type apiFunc func(w http.ResponseWriter, r *http.Request) error

func makeHttpHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			JSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		}
	}
}

func JSON(w http.ResponseWriter, status int, v any) error {
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

type APIServer struct {
	listenAdrr string
	game       *Game
}

func NewAPIServer(addr string, g *Game) *APIServer {
	return &APIServer{
		listenAdrr: addr,
		game:       g,
	}
}

func (apiServer *APIServer) Run() {
	r := mux.NewRouter()

	r.HandleFunc("/ready/{addr}", makeHttpHandleFunc(apiServer.handlePlayerReady))

	http.ListenAndServe(apiServer.listenAdrr, r)
}

func (apiServer *APIServer) handlePlayerReady(w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	apiServer.game.SetReady()
	return JSON(w, http.StatusOK, vars)
}

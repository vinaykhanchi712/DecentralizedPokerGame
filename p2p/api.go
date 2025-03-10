package p2p

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

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
	r.HandleFunc("/fold", makeHttpHandleFunc(apiServer.handlePlayerActionFold))
	r.HandleFunc("/check", makeHttpHandleFunc(apiServer.handlePlayerCheck))
	r.HandleFunc("/bet/{value}", makeHttpHandleFunc(apiServer.handlePlayerBet))

	http.ListenAndServe(apiServer.listenAdrr, r)
}

func (apiServer *APIServer) handlePlayerReady(w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	apiServer.game.SetReady()
	return JSON(w, http.StatusOK, vars)
}

func (apiServer *APIServer) handlePlayerActionFold(w http.ResponseWriter, r *http.Request) error {
	if err := apiServer.game.TakeAction(PlayerActionFold, 0); err != nil {
		return err
	}
	return JSON(w, http.StatusOK, "Folded")
}

func (apiServer *APIServer) handlePlayerCheck(w http.ResponseWriter, r *http.Request) error {
	if err := apiServer.game.TakeAction(PlayerActionCheck, 0); err != nil {
		return err
	}
	return JSON(w, http.StatusOK, "Checked")
}

func (apiServer *APIServer) handlePlayerBet(w http.ResponseWriter, r *http.Request) error {
	valueStr := mux.Vars(r)["value"]
	value, err := strconv.Atoi(valueStr)

	if err != nil {
		return err
	}

	if err := apiServer.game.TakeAction(PlayerActionBet, value); err != nil {
		return err
	}
	return JSON(w, http.StatusOK, fmt.Sprintf("bet palced with value %d", value))
}

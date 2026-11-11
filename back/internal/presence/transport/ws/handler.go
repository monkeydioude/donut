// Package wstransport is the presence module's websocket: GET /api/ws?player={uuid}&name={name}&robot={slug}. One
// socket for the game: the server puts the player on their map, and moves them from map to map (ADR 0012).
package wstransport

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"

	"donut/internal/presence/app"
	"donut/internal/presence/domain"
)

const (
	// maxMessageBytes bounds what a client sends: a bigger message closes its connection.
	maxMessageBytes = 1024
	// loggedBytes is how much of an ignored message is logged.
	loggedBytes = 200
)

// Hub connects a player to the map they're on.
type Hub interface {
	Connect(ctx context.Context, who domain.Profile, client app.Client) (*app.Session, error)
}

type Handler struct {
	hub Hub
	// originPatterns are the hosts browsers may open the websocket from, besides the API's own.
	originPatterns []string
	log            *slog.Logger
}

func NewHandler(hub Hub, originPatterns []string, log *slog.Logger) *Handler {
	return &Handler{hub: hub, originPatterns: originPatterns, log: log.With("component", "ws")}
}

// ServeHTTP checks the player before upgrading, then puts them on their map and passes their requests on until they
// leave. The logs name players by their public id: the guest id is a secret.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := h.log.With("remote", r.RemoteAddr)
	log.DebugContext(r.Context(), "ws connecting", "origin", r.Header.Get("Origin"))
	who, err := playerFrom(r)
	if err != nil {
		log.DebugContext(r.Context(), "ws refused: bad player", "status", http.StatusBadRequest, "reason", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log = log.With("player", who.ID)
	// Chunks are big and repetitive: messages over 512 bytes are compressed, each on its own (ADR 0003).
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: h.originPatterns, CompressionMode: websocket.CompressionNoContextTakeover})
	if err != nil {
		log.DebugContext(r.Context(), "ws refused by the upgrade (foreign origin?)", "origin", r.Header.Get("Origin"), "error", err)
		return // Accept answered already: 403 for a foreign origin
	}
	log.DebugContext(r.Context(), "ws open", "name", who.Name, "robot", who.Robot, "compression_offered", r.Header.Get("Sec-WebSocket-Extensions") != "")
	conn.SetReadLimit(maxMessageBytes)
	c := newClient(conn, log)
	defer c.Close()
	session, err := h.hub.Connect(r.Context(), who, c)
	if err != nil {
		log.ErrorContext(r.Context(), "ws join failed", "error", err)
		return
	}
	defer session.Leave()
	start := time.Now()
	reason := readRequests(r.Context(), conn, session, log)
	log.DebugContext(r.Context(), "ws closed", "close_status", websocket.CloseStatus(reason), "reason", reason, "lasted", time.Since(start))
}

// playerFrom reads the player's guest id, name and robot. It gives the public id: the guest id goes no further.
func playerFrom(r *http.Request) (domain.Profile, error) {
	query := r.URL.Query()
	guestID, err := domain.ParseGuestID(query.Get("player"))
	if err != nil {
		return domain.Profile{}, err
	}
	name, err := domain.CleanName(query.Get("name"))
	if err != nil {
		return domain.Profile{}, err
	}
	robot, err := domain.CleanRobot(query.Get("robot"))
	if err != nil {
		return domain.Profile{}, err
	}
	return domain.Profile{ID: domain.PublicID(guestID), Name: name, Robot: robot}, nil
}

// readRequests passes every request the client sends to their session, until the connection ends, and gives why it
// ended. Other messages are ignored.
func readRequests(ctx context.Context, conn *websocket.Conn, session *app.Session, log *slog.Logger) error {
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return err
		}
		req, ok := decodeRequest(data)
		if !ok {
			log.DebugContext(ctx, "ws message ignored: not a plan, walk, teleport, enter or exit", "bytes", len(data), "message", string(data[:min(len(data), loggedBytes)]))
			continue
		}
		log.DebugContext(ctx, "ws receive", "request", req.kind, "to", req.to, "bytes", len(data))
		switch req.kind {
		case "plan":
			session.Plan(req.to)
		case "walk":
			session.Walk(req.to)
		case "teleport":
			session.Teleport(ctx, req.toMap, req.to)
		case "enter":
			session.Enter(ctx, req.to)
		case "exit":
			session.Exit(ctx, req.to)
		}
	}
}

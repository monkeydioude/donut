// Package wstransport is the presence module's websocket: GET /api/maps/{slug}/ws?player={uuid}&name={name}&robot={slug}.
package wstransport

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"

	"donut/internal/presence/app"
	"donut/internal/presence/domain"
	"donut/pkg/errd"
)

const (
	// maxMessageBytes bounds what a client sends: a bigger message closes its connection.
	maxMessageBytes = 1024
	// loggedBytes is how much of an ignored message is logged.
	loggedBytes = 200
)

// Rooms finds the room of a map.
type Rooms interface {
	Room(ctx context.Context, slug string) (*app.Room, error)
}

type Handler struct {
	rooms Rooms
	// originPatterns are the hosts browsers may open the websocket from, besides the API's own.
	originPatterns []string
	log            *slog.Logger
}

func NewHandler(rooms Rooms, originPatterns []string, log *slog.Logger) *Handler {
	return &Handler{rooms: rooms, originPatterns: originPatterns, log: log.With("component", "ws")}
}

// ServeHTTP checks the player and the map before upgrading, then passes their requests to the room until they leave.
// The logs name players by their public id: the guest id is a secret.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	log := h.log.With("map", slug, "remote", r.RemoteAddr)
	log.DebugContext(r.Context(), "ws connecting", "origin", r.Header.Get("Origin"))
	who, err := playerFrom(r)
	if err != nil {
		log.DebugContext(r.Context(), "ws refused: bad player", "status", http.StatusBadRequest, "reason", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id := who.ID
	log = log.With("player", id)
	room, err := h.rooms.Room(r.Context(), slug)
	if err != nil {
		writeRoomError(w, r, log, err)
		return
	}
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
	if err := room.Join(who, c); err != nil {
		log.ErrorContext(r.Context(), "ws join failed", "error", err)
		return
	}
	defer room.Leave(id, c)
	start := time.Now()
	reason := readRequests(r.Context(), conn, room, id, c, log)
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

func writeRoomError(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error) {
	if errors.Is(err, errd.ErrNotFound) {
		log.DebugContext(r.Context(), "ws refused: unknown map", "status", http.StatusNotFound)
		http.Error(w, "map not found", http.StatusNotFound)
		return
	}
	log.ErrorContext(r.Context(), "ws refused: room", "status", http.StatusInternalServerError, "error", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

// readRequests passes every plan and walk the client sends to the room, until the connection ends,
// and gives why it ended. Other messages are ignored.
func readRequests(ctx context.Context, conn *websocket.Conn, room *app.Room, id string, c app.Client, log *slog.Logger) error {
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return err
		}
		req, ok := decodeRequest(data)
		if !ok {
			log.DebugContext(ctx, "ws message ignored: not a plan or walk", "bytes", len(data), "message", string(data[:min(len(data), loggedBytes)]))
			continue
		}
		log.DebugContext(ctx, "ws receive", "request", req.kind, "to", req.to, "bytes", len(data))
		switch req.kind {
		case "plan":
			room.Plan(id, c, req.to)
		case "walk":
			room.Walk(id, c, req.to)
		}
	}
}

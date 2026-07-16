package collaboration

import (
	"fmt"
	"sync"
)

type EventType string

const (
	PlaylistUpdated EventType = "playlist_updated"
	TasteShared     EventType = "taste_shared"
	FriendAdded     EventType = "friend_added"
)

type Event struct {
	Type    EventType   `json:"type"`
	Payload interface{} `json:"payload"`
	UserID  string      `json:"user_id"`
}

type Hub struct {
	mu        sync.RWMutex
	rooms     map[string][]chan Event
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string][]chan Event),
	}
}

func (h *Hub) JoinRoom(roomID string, ch chan Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.rooms[roomID] = append(h.rooms[roomID], ch)
}

func (h *Hub) LeaveRoom(roomID string, ch chan Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	channels := h.rooms[roomID]
	for i, c := range channels {
		if c == ch {
			h.rooms[roomID] = append(channels[:i], channels[i+1:]...)
			break
		}
	}
}

func (h *Hub) Emit(roomID string, event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, ch := range h.rooms[roomID] {
		select {
		case ch <- event:
		default:
		}
	}
}

func (h *Hub) RoomSize(roomID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[roomID])
}

type CollaborationService struct {
	Hub *Hub
}

func NewCollaborationService() *CollaborationService {
	return &CollaborationService{
		Hub: NewHub(),
	}
}

func (s *CollaborationService) PlaylistRoomID(playlistID string) string {
	return fmt.Sprintf("playlist:%s", playlistID)
}

func (s *CollaborationService) UserRoomID(userID string) string {
	return fmt.Sprintf("user:%s", userID)
}

func (s *CollaborationService) NotifyPlaylistUpdate(playlistID string, userID string, payload interface{}) {
	s.Hub.Emit(s.PlaylistRoomID(playlistID), Event{
		Type:    PlaylistUpdated,
		Payload: payload,
		UserID:  userID,
	})
}

func (s *CollaborationService) NotifyTasteShare(userID string, payload interface{}) {
	s.Hub.Emit(s.UserRoomID(userID), Event{
		Type:    TasteShared,
		Payload: payload,
		UserID:  userID,
	})
}

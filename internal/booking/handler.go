package booking

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"booking-system/internal/utils"
)

type Handler struct {
	svc *Service
}

func NewHandler(x *Service) *Handler {
	return &Handler{svc: x}
}

type HoldRequest struct {
	UserID string `json:"user_id"`
}

type holdSeatRequest struct {
	UserID string `json:"user_id"`
}

type SeatInfo struct {
	SeatID    string `json:"seat_id"`
	UserID    string `json:"user_id"`
	Booked    bool   `json:"booked"`
	Confirmed bool   `json:"confirmed"`
}

type HoldResponse struct {
	SessionID string `json:"session_id"`
	MovieID   string `json:"movieID"`
	SeatID    string `json:"seat_id"`
	ExpiresAt string `json:"expires_at"`
}
type SessionResponse struct {
	SessionID string `json:"session_id"`
	MovieID   string `json:"movie_id"`
	SeatID    string `json:"seat_id"`
	UserID    string `json:"user_id"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

func (h *Handler) ConfirmSeat(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionID")
	var req holdSeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return
	}
	if req.UserID == "" {
		return
	}
	session, err := h.svc.ConfirmSeat(r.Context(), sessionID, req.UserID)
	if err != nil {
		return
	}
	utils.WriteJSON(w, http.StatusOK, SessionResponse{
		SessionID: session.ID,
		MovieID:   session.MovieID,
		SeatID:    session.SeatID,
		UserID:    req.UserID,
		Status:    session.Status,
	})
}

func (h *Handler) ReleaseSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionID")
	var req holdSeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println(err)
		return
	}
	if req.UserID == "" {
		return
	}
	err := h.svc.ReleaseSeat(r.Context(), sessionID, req.UserID)
	if err != nil {
		log.Println(err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) HoldSeat(w http.ResponseWriter, r *http.Request) {
	moviedID := r.PathValue("movieID")
	seatID := r.PathValue("seatID")
	var req HoldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Println(err)
		utils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}
	data := Booking{
		UserID:  req.UserID,
		MovieID: moviedID,
		SeatID:  seatID,
	}
	session, err := h.svc.Book(data)
	if err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Failed to book seat"})
		return
	}
	utils.WriteJSON(w, http.StatusOK, HoldResponse{
		SeatID:    seatID,
		MovieID:   session.MovieID,
		SessionID: session.ID,
		ExpiresAt: session.ExpiresAt.Format(time.RFC3339),
	})
}

func (h *Handler) ListSeats(w http.ResponseWriter, r *http.Request) {
	movieID := r.PathValue("movieID")
	bookings := h.svc.ListBooking(r.Context(), movieID)
	seats := make([]SeatInfo, 0, len(bookings))
	for _, b := range bookings {
		seats = append(seats, SeatInfo{
			SeatID:    b.SeatID,
			UserID:    b.UserID,
			Booked:    true,
			Confirmed: b.Status == "confirmed",
		})
	}
	utils.WriteJSON(w, http.StatusOK, seats)
}

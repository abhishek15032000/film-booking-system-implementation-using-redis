package main

import (
	"log"
	"net/http"

	"booking-system/internal/adapters"
	"booking-system/internal/booking"
	"booking-system/internal/utils"
)

type MovieResponse struct {
	ID          string `json:"id,omitempty"`
	Title       string `json:"title,omitempty"`
	Rows        int    `json:"rows,omitempty"`
	SeatsPerRow int    `json:"seats_per_row,omitempty"`
}

var movies = []MovieResponse{
	{ID: "Inception", Title: "Inception", Rows: 5, SeatsPerRow: 8},
	{ID: "F1", Title: "F1", Rows: 4, SeatsPerRow: 6},
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /movies", ListMovies)
	mux.Handle("GET /", http.FileServer(http.Dir("static"))) // SERVE HTML FILE.
	// 3 new end points
	store := booking.NewRedisStore(adapters.NewClient("localhost:6379"))
	svc := booking.NewService(store)
	bookingHandler := booking.NewHandler(svc)
	mux.HandleFunc("GET /movies/{movieID}/seats", bookingHandler.ListSeats)
	mux.HandleFunc("POST /movies/{movieID}/seats/{seatID}/hold", bookingHandler.HoldSeat)
	mux.HandleFunc("PUT /sessions/{sessionID}/confirm", bookingHandler.ConfirmSeat)
	mux.HandleFunc("DELETE /sessions/{sessionID}", bookingHandler.ReleaseSession)

	log.Println("Server Starting Up At Port:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic("Failed to start the server")
	}
}

func ListMovies(w http.ResponseWriter, r *http.Request) {
	utils.WriteJSON(w, http.StatusOK, movies)
}

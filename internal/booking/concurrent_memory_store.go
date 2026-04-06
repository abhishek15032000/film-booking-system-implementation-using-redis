package booking

import "sync"

// pesimmistic locking

type ConcurrentMemoryStore struct {
	mu       sync.RWMutex
	bookings map[string]Booking
}

func NewConcurrentMemoryStore() *ConcurrentMemoryStore {
	return &ConcurrentMemoryStore{
		bookings: map[string]Booking{},
	}
}

func (s *ConcurrentMemoryStore) Book(b Booking) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.bookings[b.SeatID]; exists {
		// then we cant book
		return ErrSeatAlreadyBooked
	}
	s.bookings[b.SeatID] = b
	return nil
}
func (s *ConcurrentMemoryStore) ListBookings(movieID string) []Booking {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result = make([]Booking, 0)
	for _, b := range s.bookings {
		if b.MovieID == movieID {
			result = append(result, b)
		}
	}
	return result
}

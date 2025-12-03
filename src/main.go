package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

type Booking struct {
	ID           string  `json:"id"`
	CheckInDate  string  `json:"checkInDate"`
	CheckOutDate string  `json:"checkOutDate"`
	Guests       int     `json:"guests"`
	Price        float64 `json:"price"`
	Status       string  `json:"status"`
}

type BookingCreate struct {
	CheckInDate  string  `json:"checkInDate"`
	CheckOutDate string  `json:"checkOutDate"`
	Guests       int     `json:"guests"`
	Price        float64 `json:"price"`
}

type BookingUpdate struct {
	CheckInDate  *string  `json:"checkInDate,omitempty"`
	CheckOutDate *string  `json:"checkOutDate,omitempty"`
	Guests       *int     `json:"guests,omitempty"`
	Price        *float64 `json:"price,omitempty"`
	Status       *string  `json:"status,omitempty"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type BookingStore struct {
	mu    sync.RWMutex
	data  map[string]Booking
	order []string
}

func NewBookingStore() *BookingStore {
	return &BookingStore{
		data: make(map[string]Booking),
	}
}

func (s *BookingStore) Seed() {
	s.Add(Booking{
		ID:           newUUID(),
		CheckInDate:  "2025-12-20",
		CheckOutDate: "2025-12-25",
		Guests:       2,
		Price:        450.00,
		Status:       "confirmed",
	})
	s.Add(Booking{
		ID:           newUUID(),
		CheckInDate:  "2025-11-10",
		CheckOutDate: "2025-11-12",
		Guests:       1,
		Price:        199.99,
		Status:       "pending",
	})
}

func (s *BookingStore) Add(b Booking) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[b.ID] = b
	s.order = append(s.order, b.ID)
}

func (s *BookingStore) Update(b Booking) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[b.ID]; !ok {
		return false
	}
	s.data[b.ID] = b
	return true
}

func (s *BookingStore) Get(id string) (Booking, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.data[id]
	return b, ok
}

func (s *BookingStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[id]; !ok {
		return false
	}
	delete(s.data, id)
	for i, existing := range s.order {
		if existing == id {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	return true
}

func (s *BookingStore) List(offset, limit int) []Booking {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if offset >= len(s.order) {
		return []Booking{}
	}
	end := offset + limit
	if end > len(s.order) {
		end = len(s.order)
	}
	result := make([]Booking, 0, end-offset)
	for _, id := range s.order[offset:end] {
		if b, ok := s.data[id]; ok {
			result = append(result, b)
		}
	}
	return result
}

type Server struct {
	store *BookingStore
}

func NewServer() *Server {
	store := NewBookingStore()
	store.Seed()
	return &Server{store: store}
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/bookings", s.handleBookings)
	mux.HandleFunc("/bookings/", s.handleBookingByID)
	return loggingMiddleware(mux)
}

func (s *Server) handleBookings(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/bookings" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	switch r.Method {
	case http.MethodPost:
		s.createBooking(w, r)
	case http.MethodGet:
		s.listBookings(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleBookingByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/bookings/")
	segments := strings.SplitN(path, "/", 2)
	if len(segments[0]) == 0 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	id := segments[0]
	isCancel := len(segments) == 2 && segments[1] == "cancel"

	switch {
	case isCancel && r.Method == http.MethodPost:
		s.cancelBooking(w, r, id)
	case !isCancel:
		s.bookingResource(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) bookingResource(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		s.getBooking(w, r, id)
	case http.MethodPut:
		s.replaceBooking(w, r, id)
	case http.MethodPatch:
		s.updateBooking(w, r, id)
	case http.MethodDelete:
		s.deleteBooking(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) createBooking(w http.ResponseWriter, r *http.Request) {
	var payload BookingCreate
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateCreate(payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	booking := Booking{
		ID:           newUUID(),
		CheckInDate:  payload.CheckInDate,
		CheckOutDate: payload.CheckOutDate,
		Guests:       payload.Guests,
		Price:        payload.Price,
		Status:       "confirmed",
	}
	s.store.Add(booking)
	writeJSON(w, http.StatusCreated, booking)
}

func (s *Server) listBookings(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	items := s.store.List(offset, limit)
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) getBooking(w http.ResponseWriter, r *http.Request, id string) {
	booking, ok := s.store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "booking not found")
		return
	}
	writeJSON(w, http.StatusOK, booking)
}

func (s *Server) replaceBooking(w http.ResponseWriter, r *http.Request, id string) {
	existing, ok := s.store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "booking not found")
		return
	}
	var payload BookingCreate
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateCreate(payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	updated := Booking{
		ID:           id,
		CheckInDate:  payload.CheckInDate,
		CheckOutDate: payload.CheckOutDate,
		Guests:       payload.Guests,
		Price:        payload.Price,
		Status:       existing.Status,
	}
	s.store.Update(updated)
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) updateBooking(w http.ResponseWriter, r *http.Request, id string) {
	current, ok := s.store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "booking not found")
		return
	}
	var payload BookingUpdate
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if payload.CheckInDate == nil && payload.CheckOutDate == nil && payload.Guests == nil && payload.Price == nil && payload.Status == nil {
		writeError(w, http.StatusBadRequest, "no fields provided for update")
		return
	}
	if payload.CheckInDate != nil {
		current.CheckInDate = *payload.CheckInDate
	}
	if payload.CheckOutDate != nil {
		current.CheckOutDate = *payload.CheckOutDate
	}
	if payload.Guests != nil {
		if *payload.Guests < 1 {
			writeError(w, http.StatusBadRequest, "guests must be at least 1")
			return
		}
		current.Guests = *payload.Guests
	}
	if payload.Price != nil {
		if *payload.Price < 0 {
			writeError(w, http.StatusBadRequest, "price must be non-negative")
			return
		}
		current.Price = *payload.Price
	}
	if payload.Status != nil {
		current.Status = *payload.Status
	}
	s.store.Update(current)
	writeJSON(w, http.StatusOK, current)
}

func (s *Server) deleteBooking(w http.ResponseWriter, r *http.Request, id string) {
	if ok := s.store.Delete(id); !ok {
		writeError(w, http.StatusNotFound, "booking not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) cancelBooking(w http.ResponseWriter, r *http.Request, id string) {
	booking, ok := s.store.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "booking not found")
		return
	}
	booking.Status = "cancelled"
	s.store.Update(booking)
	writeJSON(w, http.StatusOK, booking)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{
		Code:    status,
		Message: msg,
	})
}

func decodeJSON(r *http.Request, dst interface{}) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}
	return nil
}

func validateCreate(payload BookingCreate) error {
	if payload.CheckInDate == "" || payload.CheckOutDate == "" {
		return fmt.Errorf("checkInDate and checkOutDate are required")
	}
	if payload.Guests < 1 {
		return fmt.Errorf("guests must be at least 1")
	}
	if payload.Price < 0 {
		return fmt.Errorf("price must be non-negative")
	}
	return nil
}

func parsePagination(r *http.Request) (int, int) {
	const (
		defaultLimit = 20
		maxLimit     = 100
	)
	limit := defaultLimit
	offset := 0

	if raw := r.URL.Query().Get("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			if v > maxLimit {
				v = maxLimit
			}
			limit = v
		}
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v >= 0 {
			offset = v
		}
	}
	return limit, offset
}

func newUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]))
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func main() {
	server := NewServer()
	port := os.Getenv("PORT")
	if port == "" {
		port = "7070"
	}
	addr := ":" + port
	log.Printf("Mock bookings server listening on %s", addr)
	if err := http.ListenAndServe(addr, server.routes()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

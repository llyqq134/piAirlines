package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func (s *Server) mountDomain(rg *gin.RouterGroup) {
	admin := rg.Group("")
	admin.Use(RequireRole("admin"))

	// reference data
	rg.GET("/airlines", s.listAirlines)
	admin.POST("/airlines", s.createAirline)
	admin.PUT("/airlines/:id", s.updateAirline)
	admin.DELETE("/airlines/:id", s.deleteAirline)

	rg.GET("/tariffs", s.listTariffs)
	admin.POST("/tariffs", s.createTariff)
	admin.PUT("/tariffs/:id", s.updateTariff)
	admin.DELETE("/tariffs/:id", s.deleteTariff)

	// flights
	rg.GET("/flights", s.listFlights)
	admin.POST("/flights", s.createFlight)
	admin.PUT("/flights/:id", s.updateFlight)
	admin.DELETE("/flights/:id", s.deleteFlight)

	// passengers
	rg.GET("/passengers", s.listPassengers)
	admin.POST("/passengers", s.createPassenger)
	admin.PUT("/passengers/:id", s.updatePassenger)
	admin.DELETE("/passengers/:id", s.deletePassenger)

	// bookings / sales / refunds
	rg.GET("/bookings", s.listBookings)
	rg.POST("/bookings", s.createBooking)

	rg.GET("/sales", s.listSales)
	rg.POST("/sales", s.createSale)

	rg.GET("/refunds", s.listRefunds)
	rg.POST("/refunds", s.createRefund)

	// notifications / reports
	rg.GET("/notifications", s.listNotifications)
	rg.GET("/reports", s.listReports)
	admin.POST("/reports", s.createReport)
	rg.GET("/reports/sales-summary", s.salesSummary)
}

func (s *Server) listAirlines(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	rows, err := s.DB.Query(ctx, `SELECT id, name, code FROM airlines ORDER BY id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer rows.Close()
	var out []gin.H
	for rows.Next() {
		var id int
		var name, code string
		if err := rows.Scan(&id, &name, &code); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		out = append(out, gin.H{"id": id, "name": name, "code": code})
	}
	c.JSON(http.StatusOK, out)
}

type createAirlineReq struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

func (s *Server) createAirline(c *gin.Context) {
	var req createAirlineReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" || req.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and code required"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var id int
	err := s.DB.QueryRow(ctx, `INSERT INTO airlines(name, code) VALUES ($1, $2) RETURNING id`, req.Name, req.Code).Scan(&id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "insert failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id})
}

func (s *Server) updateAirline(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req createAirlineReq
	if id <= 0 || c.ShouldBindJSON(&req) != nil || req.Name == "" || req.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	ct, err := s.DB.Exec(ctx, `UPDATE airlines SET name=$1, code=$2 WHERE id=$3`, req.Name, req.Code, id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "update failed"})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) deleteAirline(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	ct, err := s.DB.Exec(ctx, `DELETE FROM airlines WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "delete failed"})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) listTariffs(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	rows, err := s.DB.Query(ctx, `SELECT id, name, price::text FROM tariffs ORDER BY id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer rows.Close()
	var out []gin.H
	for rows.Next() {
		var id int
		var name string
		var price string
		if err := rows.Scan(&id, &name, &price); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		out = append(out, gin.H{"id": id, "name": name, "price": price})
	}
	c.JSON(http.StatusOK, out)
}

type createTariffReq struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func (s *Server) createTariff(c *gin.Context) {
	var req createTariffReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" || req.Price < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and non-negative price required"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var id int
	err := s.DB.QueryRow(ctx, `INSERT INTO tariffs(name, price) VALUES ($1, $2) RETURNING id`, req.Name, req.Price).Scan(&id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "insert failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id})
}

func (s *Server) updateTariff(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req createTariffReq
	if id <= 0 || c.ShouldBindJSON(&req) != nil || req.Name == "" || req.Price < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	ct, err := s.DB.Exec(ctx, `UPDATE tariffs SET name=$1, price=$2 WHERE id=$3`, req.Name, req.Price, id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "update failed"})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) deleteTariff(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	ct, err := s.DB.Exec(ctx, `DELETE FROM tariffs WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "delete failed"})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) listFlights(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	rows, err := s.DB.Query(ctx, `
		SELECT f.id, f.flight_number, f.airline_id, a.code, f.tariff_id, t.price::text,
		       COALESCE(se.total_seats, 0), COALESCE(se.available_seats, 0),
		       f.origin, f.destination, f.departure_time, f.arrival_time,
		       f.is_completed, f.completed_at
		FROM flights f
		JOIN airlines a ON a.id=f.airline_id
		JOIN tariffs t ON t.id=f.tariff_id
		LEFT JOIN seats se ON se.flight_id=f.id
		ORDER BY f.id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer rows.Close()
	var out []gin.H
	for rows.Next() {
		var id, airlineID, tariffID int
		var num, airlineCode string
		var price string
		var total, avail int
		var origin, destination string
		var departureAt, arrivalAt *time.Time
		var isCompleted bool
		var completedAt *time.Time
		if err := rows.Scan(&id, &num, &airlineID, &airlineCode, &tariffID, &price, &total, &avail, &origin, &destination, &departureAt, &arrivalAt, &isCompleted, &completedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		out = append(out, gin.H{
			"id": id, "flight_number": num,
			"airline": gin.H{"id": airlineID, "code": airlineCode},
			"tariff":  gin.H{"id": tariffID, "price": price},
			"seats":   gin.H{"total": total, "available": avail},
			"origin":  origin, "destination": destination,
			"departure_time": departureAt, "arrival_time": arrivalAt,
			"is_completed": isCompleted, "completed_at": completedAt,
		})
	}
	c.JSON(http.StatusOK, out)
}

type createFlightReq struct {
	FlightNumber   string     `json:"flight_number"`
	AirlineID      int        `json:"airline_id"`
	TariffID       int        `json:"tariff_id"`
	TotalSeats     int        `json:"total_seats"`
	AvailableSeats int        `json:"available_seats"`
	Origin         string     `json:"origin"`
	Destination    string     `json:"destination"`
	DepartureTime  *time.Time `json:"departure_time"`
	ArrivalTime    *time.Time `json:"arrival_time"`
}

func (s *Server) createFlight(c *gin.Context) {
	var req createFlightReq
	if err := c.ShouldBindJSON(&req); err != nil || req.FlightNumber == "" || req.AirlineID <= 0 || req.TariffID <= 0 || req.TotalSeats < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if req.AvailableSeats == 0 && req.TotalSeats > 0 {
		req.AvailableSeats = req.TotalSeats
	}
	if req.AvailableSeats < 0 || req.AvailableSeats > req.TotalSeats {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid available_seats"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx begin failed"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id int
	origin := req.Origin
	if origin == "" {
		origin = "Не указано"
	}
	destination := req.Destination
	if destination == "" {
		destination = "Не указано"
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO flights(flight_number, airline_id, tariff_id, origin, destination, departure_time, arrival_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, req.FlightNumber, req.AirlineID, req.TariffID, origin, destination, req.DepartureTime, req.ArrivalTime).Scan(&id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "insert failed"})
		return
	}
	_, err = tx.Exec(ctx, `INSERT INTO seats(flight_id, total_seats, available_seats) VALUES ($1, $2, $3)`, id, req.TotalSeats, req.AvailableSeats)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "seats insert failed"})
		return
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO seat_units(flight_id, seat_no, status)
		SELECT $1, 'S' || lpad(gs::text, 3, '0'), 'available'
		FROM generate_series(1, $2) gs
	`, id, req.TotalSeats)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "seat_units insert failed"})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id})
}

func (s *Server) updateFlight(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req createFlightReq
	if id <= 0 || c.ShouldBindJSON(&req) != nil || req.FlightNumber == "" || req.AirlineID <= 0 || req.TariffID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	ct, err := s.DB.Exec(ctx, `
		UPDATE flights
		SET flight_number=$1, airline_id=$2, tariff_id=$3,
		    origin=COALESCE(NULLIF($4, ''), origin),
		    destination=COALESCE(NULLIF($5, ''), destination),
		    departure_time=COALESCE($6, departure_time),
		    arrival_time=COALESCE($7, arrival_time)
		WHERE id=$8
	`, req.FlightNumber, req.AirlineID, req.TariffID, req.Origin, req.Destination, req.DepartureTime, req.ArrivalTime, id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "update failed"})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) deleteFlight(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	ct, err := s.DB.Exec(ctx, `DELETE FROM flights WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "delete failed"})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) listPassengers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	rows, err := s.DB.Query(ctx, `SELECT id, full_name, passport FROM passengers ORDER BY id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer rows.Close()
	var out []gin.H
	for rows.Next() {
		var id int
		var name, passport string
		if err := rows.Scan(&id, &name, &passport); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		out = append(out, gin.H{"id": id, "full_name": name, "passport": passport})
	}
	c.JSON(http.StatusOK, out)
}

type createPassengerReq struct {
	FullName string `json:"full_name"`
	Passport string `json:"passport"`
}

func (s *Server) createPassenger(c *gin.Context) {
	var req createPassengerReq
	if err := c.ShouldBindJSON(&req); err != nil || req.FullName == "" || req.Passport == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "full_name and passport required"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var id int
	err := s.DB.QueryRow(ctx, `INSERT INTO passengers(full_name, passport) VALUES ($1, $2) RETURNING id`, req.FullName, req.Passport).Scan(&id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "insert failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id})
}

func (s *Server) updatePassenger(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req createPassengerReq
	if id <= 0 || c.ShouldBindJSON(&req) != nil || req.FullName == "" || req.Passport == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	ct, err := s.DB.Exec(ctx, `UPDATE passengers SET full_name=$1, passport=$2 WHERE id=$3`, req.FullName, req.Passport, id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "update failed"})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) deletePassenger(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	ct, err := s.DB.Exec(ctx, `DELETE FROM passengers WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "delete failed"})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) listBookings(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	rows, err := s.DB.Query(ctx, `
		SELECT b.id, b.passenger_id, p.full_name, b.flight_id, f.flight_number, b.created_at,
		       COALESCE(su.seat_no, '')
		FROM bookings b
		JOIN passengers p ON p.id=b.passenger_id
		JOIN flights f ON f.id=b.flight_id
		LEFT JOIN seat_units su ON su.booking_id=b.id
		ORDER BY b.id DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer rows.Close()
	var out []gin.H
	for rows.Next() {
		var id, pid, fid int
		var pname, fnum string
		var created time.Time
		var seatNo string
		if err := rows.Scan(&id, &pid, &pname, &fid, &fnum, &created, &seatNo); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		out = append(out, gin.H{
			"id":         id,
			"passenger":  gin.H{"id": pid, "full_name": pname},
			"flight":     gin.H{"id": fid, "flight_number": fnum},
			"created_at": created,
			"seat_no":    seatNo,
		})
	}
	c.JSON(http.StatusOK, out)
}

type createBookingReq struct {
	PassengerID int `json:"passenger_id"`
	FlightID    int `json:"flight_id"`
}

func (s *Server) createBooking(c *gin.Context) {
	var req createBookingReq
	if err := c.ShouldBindJSON(&req); err != nil || req.PassengerID <= 0 || req.FlightID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "passenger_id and flight_id required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx begin failed"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var available int
	err = tx.QueryRow(ctx, `SELECT available_seats FROM seats WHERE flight_id=$1 FOR UPDATE`, req.FlightID).Scan(&available)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "flight seats not found"})
		return
	}
	if available <= 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "no seats available"})
		return
	}

	var seatUnitID int64
	var seatNo string
	err = tx.QueryRow(ctx, `
		SELECT id, seat_no
		FROM seat_units
		WHERE flight_id=$1 AND status='available' AND booking_id IS NULL
		ORDER BY id
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`, req.FlightID).Scan(&seatUnitID, &seatNo)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "no seat units available"})
		return
	}

	var bookingID int
	err = tx.QueryRow(ctx, `INSERT INTO bookings(passenger_id, flight_id) VALUES ($1, $2) RETURNING id`, req.PassengerID, req.FlightID).Scan(&bookingID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "booking insert failed"})
		return
	}
	_, err = tx.Exec(ctx, `UPDATE seat_units SET status='held', booking_id=$1 WHERE id=$2`, bookingID, seatUnitID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seat assign failed"})
		return
	}
	_, err = tx.Exec(ctx, `UPDATE seats SET available_seats=available_seats-1 WHERE flight_id=$1`, req.FlightID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seats update failed"})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": bookingID, "seat_no": seatNo})
}

func (s *Server) listSales(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	rows, err := s.DB.Query(ctx, `
		SELECT s.id, s.booking_id, s.amount::text, s.created_at
		FROM sales s
		ORDER BY s.id DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer rows.Close()
	var out []gin.H
	for rows.Next() {
		var id, bid int
		var amount string
		var created time.Time
		if err := rows.Scan(&id, &bid, &amount, &created); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		out = append(out, gin.H{"id": id, "booking_id": bid, "amount": amount, "created_at": created})
	}
	c.JSON(http.StatusOK, out)
}

type createSaleReq struct {
	BookingID int `json:"booking_id"`
}

func (s *Server) createSale(c *gin.Context) {
	var req createSaleReq
	if err := c.ShouldBindJSON(&req); err != nil || req.BookingID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "booking_id required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx begin failed"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var flightID, passengerID int
	var price string
	err = tx.QueryRow(ctx, `
		SELECT b.flight_id, b.passenger_id, t.price::text
		FROM bookings b
		JOIN flights f ON f.id=b.flight_id
		JOIN tariffs t ON t.id=f.tariff_id
		WHERE b.id=$1
	`, req.BookingID).Scan(&flightID, &passengerID, &price)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "booking not found"})
		return
	}

	var saleID int
	err = tx.QueryRow(ctx, `INSERT INTO sales(booking_id, amount) VALUES ($1, $2) RETURNING id`, req.BookingID, price).Scan(&saleID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "sale already exists or insert failed"})
		return
	}
	_, _ = tx.Exec(ctx, `UPDATE seat_units SET status='sold' WHERE booking_id=$1`, req.BookingID)
	_, _ = tx.Exec(ctx, `INSERT INTO notifications(passenger_id, message) VALUES ($1, $2)`, passengerID, "Билет оплачен. Спасибо за покупку!")

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": saleID})
}

func (s *Server) listRefunds(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	rows, err := s.DB.Query(ctx, `SELECT id, sale_id, amount::text, created_at FROM refunds ORDER BY id DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer rows.Close()
	var out []gin.H
	for rows.Next() {
		var id, sid int
		var amount string
		var created time.Time
		if err := rows.Scan(&id, &sid, &amount, &created); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		out = append(out, gin.H{"id": id, "sale_id": sid, "amount": amount, "created_at": created})
	}
	c.JSON(http.StatusOK, out)
}

type createRefundReq struct {
	SaleID int     `json:"sale_id"`
	Amount float64 `json:"amount"`
}

func (s *Server) createRefund(c *gin.Context) {
	var req createRefundReq
	if err := c.ShouldBindJSON(&req); err != nil || req.SaleID <= 0 || req.Amount < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sale_id and non-negative amount required"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx begin failed"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var bookingID, passengerID, flightID int
	var saleAmount float64
	err = tx.QueryRow(ctx, `
		SELECT s.booking_id, b.passenger_id, b.flight_id, s.amount::float8
		FROM sales s
		JOIN bookings b ON b.id=s.booking_id
		WHERE s.id=$1
	`, req.SaleID).Scan(&bookingID, &passengerID, &flightID, &saleAmount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sale not found"})
		return
	}

	var refunded float64
	_ = tx.QueryRow(ctx, `SELECT COALESCE(SUM(amount), 0)::float8 FROM refunds WHERE sale_id=$1`, req.SaleID).Scan(&refunded)
	if refunded+req.Amount > saleAmount {
		c.JSON(http.StatusConflict, gin.H{"error": "refund exceeds sale amount"})
		return
	}

	var refundID int
	err = tx.QueryRow(ctx, `INSERT INTO refunds(sale_id, amount) VALUES ($1, $2) RETURNING id`, req.SaleID, req.Amount).Scan(&refundID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "refund insert failed"})
		return
	}

	// for simplicity: if fully refunded, return a seat
	if refunded+req.Amount == saleAmount {
		_, _ = tx.Exec(ctx, `UPDATE seat_units SET status='available', booking_id=NULL WHERE booking_id=$1`, bookingID)
		_, _ = tx.Exec(ctx, `UPDATE seats SET available_seats=available_seats+1 WHERE flight_id=$1`, flightID)
	}
	_, _ = tx.Exec(ctx, `INSERT INTO notifications(passenger_id, message) VALUES ($1, $2)`, passengerID, "Оформлен возврат по билету. Средства будут возвращены.")

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": refundID, "booking_id": bookingID})
}

func (s *Server) listNotifications(c *gin.Context) {
	passengerIDStr := c.Query("passenger_id")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	q := `SELECT id, passenger_id, message, created_at FROM notifications`
	var args []any
	if passengerIDStr != "" {
		pid, _ := strconv.Atoi(passengerIDStr)
		q += ` WHERE passenger_id=$1`
		args = append(args, pid)
	}
	q += ` ORDER BY id DESC LIMIT 200`

	rows, err := s.DB.Query(ctx, q, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer rows.Close()
	var out []gin.H
	for rows.Next() {
		var id, pid int
		var msg string
		var created time.Time
		if err := rows.Scan(&id, &pid, &msg, &created); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		out = append(out, gin.H{"id": id, "passenger_id": pid, "message": msg, "created_at": created})
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) listReports(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	includePayload := c.Query("include_payload") == "1"
	q := `SELECT id, report_type, created_at`
	if includePayload {
		q += `, payload`
	}
	q += ` FROM reports ORDER BY id DESC LIMIT 200`
	rows, err := s.DB.Query(ctx, q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer rows.Close()
	var out []gin.H
	for rows.Next() {
		var id int
		var t string
		var created time.Time
		if includePayload {
			var payload []byte
			if err := rows.Scan(&id, &t, &created, &payload); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
				return
			}
			out = append(out, gin.H{"id": id, "report_type": t, "created_at": created, "payload": jsonRaw(payload)})
			continue
		}
		if err := rows.Scan(&id, &t, &created); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		out = append(out, gin.H{"id": id, "report_type": t, "created_at": created})
	}
	c.JSON(http.StatusOK, out)
}

type jsonRaw []byte

func (j jsonRaw) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

type createReportReq struct {
	ReportType string `json:"report_type"`
}

func (s *Server) createReport(c *gin.Context) {
	var req createReportReq
	if err := c.ShouldBindJSON(&req); err != nil || req.ReportType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "report_type required"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var id int
	err := s.DB.QueryRow(ctx, `INSERT INTO reports(report_type) VALUES ($1) RETURNING id`, req.ReportType).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "insert failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id})
}

func (s *Server) salesSummary(c *gin.Context) {
	from := c.Query("from") // RFC3339
	to := c.Query("to")
	var fromT, toT time.Time
	var err error
	if from != "" {
		fromT, err = time.Parse(time.RFC3339, from)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from"})
			return
		}
	}
	if to != "" {
		toT, err = time.Parse(time.RFC3339, to)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to"})
			return
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	q := `
		SELECT COUNT(*), COALESCE(SUM(amount),0)::text
		FROM sales
		WHERE ($1::timestamptz IS NULL OR created_at >= $1)
		  AND ($2::timestamptz IS NULL OR created_at <= $2)
	`
	var count int64
	var sum string
	var p1 any = nil
	var p2 any = nil
	if !fromT.IsZero() {
		p1 = fromT
	}
	if !toT.IsZero() {
		p2 = toT
	}
	if err := s.DB.QueryRow(ctx, q, p1, p2).Scan(&count, &sum); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": count, "sum": sum})
}

var _ = errors.New // keep for future domain errors

package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func (s *Server) mountCustomer(rg *gin.RouterGroup) {
	cg := rg.Group("/customer")
	cg.Use(RequireRole("customer", "admin"))

	cg.GET("/flights", s.listFlights)
	cg.GET("/bookings", s.customerBookings)
	cg.GET("/profile", s.customerGetProfile)
	cg.PUT("/profile", s.customerUpdateProfile)
	cg.POST("/bookings", s.customerCreateBooking)
	cg.POST("/bookings/:id/pay", s.customerPayBooking)
	cg.POST("/cart/pay", s.customerPayCart)
	cg.POST("/cart/remove", s.customerRemoveFromCart)
	cg.POST("/sales/:id/refund", s.customerRefundSale)
}

func (s *Server) getPassengerIDByUser(ctx context.Context, userID int64) (int, error) {
	var pid sql.NullInt64
	var email string
	if err := s.DB.QueryRow(ctx, `SELECT email, passenger_id FROM users WHERE id=$1`, userID).Scan(&email, &pid); err != nil {
		return 0, err
	}
	if pid.Valid && pid.Int64 > 0 {
		return int(pid.Int64), nil
	}
	passport := "AUTO-LINK-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	var newPID int
	if err := s.DB.QueryRow(ctx, `
		INSERT INTO passengers(full_name, last_name, contact_number, passport)
		VALUES ($1, '', 'N/A', $2)
		RETURNING id
	`, email, passport).Scan(&newPID); err != nil {
		return 0, err
	}
	_, _ = s.DB.Exec(ctx, `UPDATE users SET passenger_id=$1 WHERE id=$2`, newPID, userID)
	return newPID, nil
}

func (s *Server) customerBookings(c *gin.Context) {
	claims := MustClaims(c)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	passengerID, err := s.getPassengerIDByUser(ctx, claims.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "passenger profile not linked"})
		return
	}

	rows, err := s.DB.Query(ctx, `
		SELECT b.id, b.flight_id, f.flight_number, b.created_at, COALESCE(su.seat_no, ''),
		       s.id, COALESCE(s.amount::text, ''), t.price::text, r.id, COALESCE(r.amount::text, ''),
		       f.is_completed
		FROM bookings b
		JOIN flights f ON f.id=b.flight_id
		JOIN tariffs t ON t.id=f.tariff_id
		LEFT JOIN seat_units su ON su.booking_id=b.id
		LEFT JOIN sales s ON s.booking_id=b.id
		LEFT JOIN LATERAL (
		    SELECT rr.id, rr.amount
		    FROM refunds rr
		    WHERE rr.sale_id=s.id
		    ORDER BY rr.id DESC
		    LIMIT 1
		) r ON true
		WHERE b.passenger_id=$1
		ORDER BY b.id DESC
	`, passengerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer rows.Close()
	var out []gin.H
	for rows.Next() {
		var bid, fid int
		var fnum, seat string
		var created time.Time
		var saleID, refundID *int
		var saleAmt, bookingAmt, refundAmt string
		var isCompleted bool
		if err := rows.Scan(&bid, &fid, &fnum, &created, &seat, &saleID, &saleAmt, &bookingAmt, &refundID, &refundAmt, &isCompleted); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		out = append(out, gin.H{
			"id": bid, "flight": gin.H{"id": fid, "flight_number": fnum}, "created_at": created, "seat_no": seat,
			"sale_id": saleID, "sale_amount": saleAmt, "booking_amount": bookingAmt, "refund_id": refundID, "refund_amount": refundAmt,
			"is_completed": isCompleted,
		})
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) customerCreateBooking(c *gin.Context) {
	claims := MustClaims(c)
	var req struct {
		FlightID int `json:"flight_id"`
	}
	if c.ShouldBindJSON(&req) != nil || req.FlightID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "flight_id required"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	passengerID, err := s.getPassengerIDByUser(ctx, claims.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "passenger profile not linked"})
		return
	}

	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx begin failed"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var available int
	err = tx.QueryRow(ctx, `SELECT available_seats FROM seats WHERE flight_id=$1 FOR UPDATE`, req.FlightID).Scan(&available)
	if err != nil || available <= 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "no seats available"})
		return
	}
	var seatUnitID int64
	var seatNo string
	err = tx.QueryRow(ctx, `
		SELECT id, seat_no FROM seat_units
		WHERE flight_id=$1 AND status='available' AND booking_id IS NULL
		ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED
	`, req.FlightID).Scan(&seatUnitID, &seatNo)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Self-heal: if seats exist but seat_units are missing, recreate missing units.
			var totalSeats int
			if scanErr := tx.QueryRow(ctx, `SELECT total_seats FROM seats WHERE flight_id=$1`, req.FlightID).Scan(&totalSeats); scanErr == nil && totalSeats > 0 {
				_, _ = tx.Exec(ctx, `
					INSERT INTO seat_units(flight_id, seat_no, status)
					SELECT $1, 'S' || lpad(gs::text, 3, '0'), 'available'
					FROM generate_series(1, $2) gs
					ON CONFLICT (flight_id, seat_no) DO NOTHING
				`, req.FlightID, totalSeats)
				err = tx.QueryRow(ctx, `
					SELECT id, seat_no FROM seat_units
					WHERE flight_id=$1 AND status='available' AND booking_id IS NULL
					ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED
				`, req.FlightID).Scan(&seatUnitID, &seatNo)
			}
		}
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "места недоступны"})
			return
		}
	}
	var bookingID int
	err = tx.QueryRow(ctx, `INSERT INTO bookings(passenger_id, flight_id) VALUES ($1, $2) RETURNING id`, passengerID, req.FlightID).Scan(&bookingID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "booking insert failed"})
		return
	}
	_, _ = tx.Exec(ctx, `UPDATE seat_units SET status='held', booking_id=$1 WHERE id=$2`, bookingID, seatUnitID)
	_, _ = tx.Exec(ctx, `UPDATE seats SET available_seats=available_seats-1 WHERE flight_id=$1`, req.FlightID)
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": bookingID, "seat_no": seatNo})
}

func (s *Server) customerPayBooking(c *gin.Context) {
	claims := MustClaims(c)
	bookingID, _ := strconv.Atoi(c.Param("id"))
	if bookingID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking id"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	passengerID, err := s.getPassengerIDByUser(ctx, claims.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "passenger profile not linked"})
		return
	}

	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx begin failed"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var ownerID int
	var price string
	err = tx.QueryRow(ctx, `
		SELECT b.passenger_id, t.price::text
		FROM bookings b
		JOIN flights f ON f.id=b.flight_id
		JOIN tariffs t ON t.id=f.tariff_id
		WHERE b.id=$1
	`, bookingID).Scan(&ownerID, &price)
	if err != nil || ownerID != passengerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "booking access denied"})
		return
	}

	var saleID int
	err = tx.QueryRow(ctx, `INSERT INTO sales(booking_id, amount) VALUES ($1, $2) RETURNING id`, bookingID, price).Scan(&saleID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "sale already exists or insert failed"})
		return
	}
	_, _ = tx.Exec(ctx, `UPDATE seat_units SET status='sold' WHERE booking_id=$1`, bookingID)
	_, _ = tx.Exec(ctx, `INSERT INTO notifications(passenger_id, message) VALUES ($1, $2)`, passengerID, "Билет оплачен. Спасибо за покупку!")
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": saleID})
}

type cartPayReq struct {
	BookingIDs []int `json:"booking_ids"`
}

func (s *Server) customerPayCart(c *gin.Context) {
	claims := MustClaims(c)
	var req cartPayReq
	if c.ShouldBindJSON(&req) != nil || len(req.BookingIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "booking_ids required"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	passengerID, err := s.getPassengerIDByUser(ctx, claims.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "passenger profile not linked"})
		return
	}

	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx begin failed"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var paidIDs []int
	for _, bookingID := range req.BookingIDs {
		var ownerID int
		var price string
		err := tx.QueryRow(ctx, `
			SELECT b.passenger_id, t.price::text
			FROM bookings b
			JOIN flights f ON f.id=b.flight_id
			JOIN tariffs t ON t.id=f.tariff_id
			WHERE b.id=$1
		`, bookingID).Scan(&ownerID, &price)
		if err != nil || ownerID != passengerID {
			continue
		}
		var saleID int
		err = tx.QueryRow(ctx, `INSERT INTO sales(booking_id, amount) VALUES ($1, $2) RETURNING id`, bookingID, price).Scan(&saleID)
		if err != nil {
			continue
		}
		_, _ = tx.Exec(ctx, `UPDATE seat_units SET status='sold' WHERE booking_id=$1`, bookingID)
		_, _ = tx.Exec(ctx, `INSERT INTO notifications(passenger_id, message) VALUES ($1, $2)`, passengerID, "Билет оплачен. Спасибо за покупку!")
		paidIDs = append(paidIDs, saleID)
	}
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"paid_sale_ids": paidIDs})
}

func (s *Server) customerRemoveFromCart(c *gin.Context) {
	claims := MustClaims(c)
	var req cartPayReq
	if c.ShouldBindJSON(&req) != nil || len(req.BookingIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "booking_ids required"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	passengerID, err := s.getPassengerIDByUser(ctx, claims.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "passenger profile not linked"})
		return
	}

	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx begin failed"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	removed := 0
	for _, bookingID := range req.BookingIDs {
		var ownerID, flightID int
		var hasSale bool
		err := tx.QueryRow(ctx, `
			SELECT b.passenger_id, b.flight_id, EXISTS(SELECT 1 FROM sales s WHERE s.booking_id=b.id)
			FROM bookings b
			WHERE b.id=$1
		`, bookingID).Scan(&ownerID, &flightID, &hasSale)
		if err != nil || ownerID != passengerID || hasSale {
			continue
		}
		_, _ = tx.Exec(ctx, `UPDATE seat_units SET status='available', booking_id=NULL WHERE booking_id=$1`, bookingID)
		_, _ = tx.Exec(ctx, `UPDATE seats SET available_seats=available_seats+1 WHERE flight_id=$1`, flightID)
		tag, err := tx.Exec(ctx, `DELETE FROM bookings WHERE id=$1`, bookingID)
		if err == nil && tag.RowsAffected() > 0 {
			removed++
		}
	}
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"removed": removed})
}

func (s *Server) customerGetProfile(c *gin.Context) {
	claims := MustClaims(c)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var out struct {
		UserID        int64  `json:"user_id"`
		Email         string `json:"email"`
		Role          string `json:"role"`
		PassengerID   int    `json:"passenger_id"`
		Name          string `json:"name"`
		LastName      string `json:"lastname"`
		ContactNumber string `json:"contact_number"`
	}
	err := s.DB.QueryRow(ctx, `
		SELECT u.id, u.email, u.role, COALESCE(u.passenger_id, 0),
		       COALESCE(p.full_name, ''), COALESCE(p.last_name, ''), COALESCE(p.contact_number, '')
		FROM users u
		LEFT JOIN passengers p ON p.id=u.passenger_id
		WHERE u.id=$1
	`, claims.UserID).Scan(&out.UserID, &out.Email, &out.Role, &out.PassengerID, &out.Name, &out.LastName, &out.ContactNumber)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		return
	}

	salesRows, err := s.DB.Query(ctx, `
		SELECT s.id, s.amount::text, s.created_at, b.id, f.flight_number
		FROM sales s
		JOIN bookings b ON b.id=s.booking_id
		JOIN flights f ON f.id=b.flight_id
		WHERE b.passenger_id=$1
		ORDER BY s.id DESC
	`, out.PassengerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sales history error"})
		return
	}
	defer salesRows.Close()
	var sales []gin.H
	for salesRows.Next() {
		var saleID, bookingID int
		var amount, flightNumber string
		var createdAt time.Time
		if err := salesRows.Scan(&saleID, &amount, &createdAt, &bookingID, &flightNumber); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "sales history scan error"})
			return
		}
		sales = append(sales, gin.H{
			"id": saleID, "amount": amount, "created_at": createdAt,
			"booking_id": bookingID, "flight_number": flightNumber,
		})
	}

	refundRows, err := s.DB.Query(ctx, `
		SELECT r.id, r.amount::text, r.created_at, s.id, b.id, f.flight_number
		FROM refunds r
		JOIN sales s ON s.id=r.sale_id
		JOIN bookings b ON b.id=s.booking_id
		JOIN flights f ON f.id=b.flight_id
		WHERE b.passenger_id=$1
		ORDER BY r.id DESC
	`, out.PassengerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "refund history error"})
		return
	}
	defer refundRows.Close()
	var refunds []gin.H
	for refundRows.Next() {
		var refundID, saleID, bookingID int
		var amount, flightNumber string
		var createdAt time.Time
		if err := refundRows.Scan(&refundID, &amount, &createdAt, &saleID, &bookingID, &flightNumber); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "refund history scan error"})
			return
		}
		refunds = append(refunds, gin.H{
			"id": refundID, "amount": amount, "created_at": createdAt,
			"sale_id": saleID, "booking_id": bookingID, "flight_number": flightNumber,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": out.UserID, "email": out.Email, "role": out.Role,
		"passenger_id": out.PassengerID, "name": out.Name, "lastname": out.LastName, "contact_number": out.ContactNumber,
		"sales_history": sales, "refund_history": refunds,
	})
}

type updateProfileReq struct {
	Name          string `json:"name"`
	LastName      string `json:"lastname"`
	ContactNumber string `json:"contact_number"`
}

func (s *Server) customerUpdateProfile(c *gin.Context) {
	claims := MustClaims(c)
	var req updateProfileReq
	if c.ShouldBindJSON(&req) != nil || req.Name == "" || req.ContactNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and contact_number required"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	pid, err := s.getPassengerIDByUser(ctx, claims.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "passenger profile not linked"})
		return
	}
	_, err = s.DB.Exec(ctx, `
		UPDATE passengers
		SET full_name=$1, last_name=$2, contact_number=$3
		WHERE id=$4
	`, req.Name, req.LastName, req.ContactNumber, pid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) customerRefundSale(c *gin.Context) {
	claims := MustClaims(c)
	saleID, _ := strconv.Atoi(c.Param("id"))
	var req struct {
		Amount float64 `json:"amount"`
	}
	if saleID <= 0 || c.ShouldBindJSON(&req) != nil || req.Amount < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	passengerID, err := s.getPassengerIDByUser(ctx, claims.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "passenger profile not linked"})
		return
	}

	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx begin failed"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var bookingID, flightID, ownerID int
	var saleAmount float64
	var isCompleted bool
	err = tx.QueryRow(ctx, `
		SELECT s.booking_id, b.flight_id, b.passenger_id, s.amount::float8, f.is_completed
		FROM sales s JOIN bookings b ON b.id=s.booking_id WHERE s.id=$1
		JOIN flights f ON f.id=b.flight_id
	`, saleID).Scan(&bookingID, &flightID, &ownerID, &saleAmount, &isCompleted)
	if err != nil || ownerID != passengerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "sale access denied"})
		return
	}
	if isCompleted {
		c.JSON(http.StatusConflict, gin.H{"error": "нельзя вернуть билет по выполненному рейсу"})
		return
	}
	var refunded float64
	_ = tx.QueryRow(ctx, `SELECT COALESCE(SUM(amount),0)::float8 FROM refunds WHERE sale_id=$1`, saleID).Scan(&refunded)
	if refunded+req.Amount > saleAmount {
		c.JSON(http.StatusConflict, gin.H{"error": "refund exceeds sale amount"})
		return
	}
	var refundID int
	err = tx.QueryRow(ctx, `INSERT INTO refunds(sale_id, amount) VALUES ($1, $2) RETURNING id`, saleID, req.Amount).Scan(&refundID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "refund insert failed"})
		return
	}
	if refunded+req.Amount == saleAmount {
		_, _ = tx.Exec(ctx, `UPDATE seat_units SET status='available', booking_id=NULL WHERE booking_id=$1`, bookingID)
		_, _ = tx.Exec(ctx, `UPDATE seats SET available_seats=available_seats+1 WHERE flight_id=$1`, flightID)
	}
	_, _ = tx.Exec(ctx, `INSERT INTO notifications(passenger_id, message) VALUES ($1, $2)`, passengerID, "Оформлен возврат по билету. Средства будут возвращены.")
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": refundID})
}

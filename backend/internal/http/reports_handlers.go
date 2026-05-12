package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) mountReports(rg *gin.RouterGroup) {
	rg.POST("/reports/generate/sales-by-airline", s.generateSalesByAirline)
	rg.POST("/reports/generate/flight-occupancy", s.generateFlightOccupancy)
}

type timeRange struct {
	From *time.Time `json:"from"`
	To   *time.Time `json:"to"`
}

func parseTimeRange(c *gin.Context) (from *time.Time, to *time.Time, ok bool) {
	fromStr := c.Query("from")
	toStr := c.Query("to")
	if fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from"})
			return nil, nil, false
		}
		from = &t
	}
	if toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to"})
			return nil, nil, false
		}
		to = &t
	}
	return from, to, true
}

func (s *Server) generateSalesByAirline(c *gin.Context) {
	from, to, ok := parseTimeRange(c)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	type row struct {
		AirlineCode string `json:"airline_code"`
		SalesCount  int64  `json:"sales_count"`
		SumAmount   string `json:"sum_amount"`
	}
	var rowsOut []row

	q := `
		SELECT a.code, COUNT(*)::bigint, COALESCE(SUM(s.amount), 0)::text
		FROM sales s
		JOIN bookings b ON b.id=s.booking_id
		JOIN flights f ON f.id=b.flight_id
		JOIN airlines a ON a.id=f.airline_id
		WHERE ($1::timestamptz IS NULL OR s.created_at >= $1)
		  AND ($2::timestamptz IS NULL OR s.created_at <= $2)
		GROUP BY a.code
		ORDER BY a.code
	`
	var p1 any = nil
	var p2 any = nil
	if from != nil {
		p1 = *from
	}
	if to != nil {
		p2 = *to
	}
	r, err := s.DB.Query(ctx, q, p1, p2)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer r.Close()
	for r.Next() {
		var rr row
		if err := r.Scan(&rr.AirlineCode, &rr.SalesCount, &rr.SumAmount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		rowsOut = append(rowsOut, rr)
	}

	payload, _ := json.Marshal(gin.H{
		"type": "sales-by-airline",
		"range": timeRange{
			From: from,
			To:   to,
		},
		"rows": rowsOut,
	})

	var reportID int
	if err := s.DB.QueryRow(ctx, `INSERT INTO reports(report_type, payload) VALUES ($1, $2) RETURNING id`, "sales-by-airline", payload).Scan(&reportID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "report insert failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": reportID, "rows": rowsOut})
}

func (s *Server) generateFlightOccupancy(c *gin.Context) {
	from, to, ok := parseTimeRange(c)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	type row struct {
		FlightID     int    `json:"flight_id"`
		FlightNumber string `json:"flight_number"`
		AirlineCode  string `json:"airline_code"`
		TotalSeats   int    `json:"total_seats"`
		Available    int    `json:"available_seats"`
		Held         int    `json:"held_seats"`
		Sold         int    `json:"sold_seats"`
		RevenueSum   string `json:"revenue_sum"`
		SalesCount   int64  `json:"sales_count"`
	}
	var rowsOut []row

	q := `
		WITH seat_stats AS (
		  SELECT flight_id,
		         COUNT(*) FILTER (WHERE status='available')::int AS available,
		         COUNT(*) FILTER (WHERE status='held')::int AS held,
		         COUNT(*) FILTER (WHERE status='sold')::int AS sold,
		         COUNT(*)::int AS total
		  FROM seat_units
		  GROUP BY flight_id
		),
		sales_stats AS (
		  SELECT b.flight_id,
		         COUNT(*)::bigint AS sales_count,
		         COALESCE(SUM(s.amount), 0)::text AS revenue_sum
		  FROM sales s
		  JOIN bookings b ON b.id=s.booking_id
		  WHERE ($1::timestamptz IS NULL OR s.created_at >= $1)
		    AND ($2::timestamptz IS NULL OR s.created_at <= $2)
		  GROUP BY b.flight_id
		)
		SELECT f.id, f.flight_number, a.code,
		       COALESCE(ss.total, 0), COALESCE(ss.available, 0), COALESCE(ss.held, 0), COALESCE(ss.sold, 0),
		       COALESCE(sa.revenue_sum, '0'), COALESCE(sa.sales_count, 0)
		FROM flights f
		JOIN airlines a ON a.id=f.airline_id
		LEFT JOIN seat_stats ss ON ss.flight_id=f.id
		LEFT JOIN sales_stats sa ON sa.flight_id=f.id
		ORDER BY f.id
	`
	var p1 any = nil
	var p2 any = nil
	if from != nil {
		p1 = *from
	}
	if to != nil {
		p2 = *to
	}
	r, err := s.DB.Query(ctx, q, p1, p2)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer r.Close()
	for r.Next() {
		var rr row
		if err := r.Scan(&rr.FlightID, &rr.FlightNumber, &rr.AirlineCode, &rr.TotalSeats, &rr.Available, &rr.Held, &rr.Sold, &rr.RevenueSum, &rr.SalesCount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		rowsOut = append(rowsOut, rr)
	}

	payload, _ := json.Marshal(gin.H{
		"type": "flight-occupancy",
		"range": timeRange{
			From: from,
			To:   to,
		},
		"rows": rowsOut,
	})
	var reportID int
	if err := s.DB.QueryRow(ctx, `INSERT INTO reports(report_type, payload) VALUES ($1, $2) RETURNING id`, "flight-occupancy", payload).Scan(&reportID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "report insert failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": reportID, "rows": rowsOut})
}

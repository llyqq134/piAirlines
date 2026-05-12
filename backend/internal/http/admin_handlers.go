package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) mountAdmin(rg *gin.RouterGroup) {
	admin := rg.Group("/admin")
	admin.Use(RequireRole("admin"))

	admin.GET("/users", s.adminListUsers)
	admin.PATCH("/users/:id/role", s.adminUpdateUserRole)
	admin.GET("/flights", s.adminListFlights)
	admin.PATCH("/flights/:id/completed", s.adminSetFlightCompleted)
}

func (s *Server) adminListUsers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	rows, err := s.DB.Query(ctx, `SELECT id, email, role, created_at FROM users ORDER BY id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer rows.Close()

	var out []gin.H
	for rows.Next() {
		var id int64
		var email, role string
		var created time.Time
		if err := rows.Scan(&id, &email, &role, &created); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		out = append(out, gin.H{"id": id, "email": email, "role": role, "created_at": created})
	}
	c.JSON(http.StatusOK, out)
}

type updateRoleReq struct {
	Role string `json:"role"`
}

func (s *Server) adminUpdateUserRole(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req updateRoleReq
	if id <= 0 || c.ShouldBindJSON(&req) != nil || (req.Role != "admin" && req.Role != "agent" && req.Role != "customer") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be admin|agent|customer"})
		return
	}

	claims := MustClaims(c)
	if claims != nil && claims.UserID == id && req.Role != "admin" {
		c.JSON(http.StatusConflict, gin.H{"error": "cannot remove own admin role"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	ct, err := s.DB.Exec(ctx, `UPDATE users SET role=$1 WHERE id=$2`, req.Role, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	if ct.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) adminListFlights(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	rows, err := s.DB.Query(ctx, `
		SELECT id, flight_number, origin, destination, departure_time, arrival_time, is_completed, completed_at
		FROM flights
		ORDER BY id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	defer rows.Close()
	var out []gin.H
	for rows.Next() {
		var id int
		var num, origin, destination string
		var dep, arr, completedAt *time.Time
		var completed bool
		if err := rows.Scan(&id, &num, &origin, &destination, &dep, &arr, &completed, &completedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan error"})
			return
		}
		out = append(out, gin.H{
			"id": id, "flight_number": num, "origin": origin, "destination": destination,
			"departure_time": dep, "arrival_time": arr,
			"is_completed": completed, "completed_at": completedAt,
		})
	}
	c.JSON(http.StatusOK, out)
}

type setFlightCompletedReq struct {
	IsCompleted bool `json:"is_completed"`
}

func (s *Server) adminSetFlightCompleted(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req setFlightCompletedReq
	if id <= 0 || c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	tag, err := s.DB.Exec(ctx, `
		UPDATE flights
		SET is_completed=$1,
		    completed_at=CASE WHEN $1 THEN now() ELSE NULL END
		WHERE id=$2
	`, req.IsCompleted, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "flight not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

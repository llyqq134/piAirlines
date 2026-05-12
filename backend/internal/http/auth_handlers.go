package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"airtickets/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func (s *Server) mountAuth(rg *gin.RouterGroup) {
	rg.POST("/auth/register", s.handleRegister)
	rg.POST("/auth/login", s.handleLogin)
	rg.POST("/auth/refresh", s.handleRefresh)
	rg.POST("/auth/logout", AuthRequired(s.Cfg), s.handleLogout)
	rg.GET("/auth/me", AuthRequired(s.Cfg), s.handleMe)

	rg.GET("/auth/oauth/google/login", s.handleGoogleLogin)
	rg.GET("/auth/oauth/google/callback", s.handleGoogleCallback)
}

type registerReq struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	Name          string `json:"name"`
	LastName      string `json:"lastname"`
	ContactNumber string `json:"contact_number"`
}

func (s *Server) handleRegister(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password required"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failed"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	role := "customer"
	var usersCount int64
	_ = s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&usersCount)
	if usersCount == 0 {
		role = "admin"
	}

	name := req.Name
	if name == "" {
		name = req.Email
	}
	lastName := req.LastName
	contact := req.ContactNumber
	if contact == "" {
		contact = "N/A"
	}
	passport := "AUTO-" + strconv.FormatInt(time.Now().UnixNano(), 10)

	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx begin failed"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var passengerID int
	err = tx.QueryRow(ctx, `INSERT INTO passengers(full_name, last_name, contact_number, passport) VALUES ($1, $2, $3, $4) RETURNING id`, name, lastName, contact, passport).Scan(&passengerID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "passport already exists"})
		return
	}

	var id int64
	err = tx.QueryRow(ctx,
		`INSERT INTO users(email, password_hash, role, passenger_id) VALUES ($1, $2, $3, $4) RETURNING id`,
		req.Email, string(hash), role, passengerID,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user already exists"})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx commit failed"})
		return
	}

	access, refresh, err := s.issueTokens(ctx, id, req.Email, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token sign failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": access, "refresh_token": refresh})
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var id int64
	var hash *string
	var role string
	err := s.DB.QueryRow(ctx, `SELECT id, password_hash, role FROM users WHERE email=$1`, req.Email).Scan(&id, &hash, &role)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if hash == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "password login disabled for this user"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*hash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	access, refresh, err := s.issueTokens(ctx, id, req.Email, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token sign failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": access, "refresh_token": refresh})
}

func (s *Server) handleMe(c *gin.Context) {
	claims := MustClaims(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no claims"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": claims.UserID, "email": claims.Email, "role": claims.Role})
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (s *Server) handleRefresh(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	userID, err := auth.ConsumeRefreshToken(ctx, s.DB, auth.HashRefreshToken(req.RefreshToken))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh invalid"})
		return
	}

	var email, role string
	if err := s.DB.QueryRow(ctx, `SELECT email, role FROM users WHERE id=$1`, userID).Scan(&email, &role); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	access, refresh, err := s.issueTokens(ctx, userID, email, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token sign failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": access, "refresh_token": refresh})
}

func (s *Server) handleLogout(c *gin.Context) {
	claims := MustClaims(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	_ = auth.RevokeAllRefreshTokens(ctx, s.DB, claims.UserID)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) issueTokens(ctx context.Context, userID int64, email, role string) (accessToken, refreshToken string, err error) {
	accessToken, err = auth.SignJWT(s.Cfg.JWTIssuer, s.Cfg.JWTSecret, s.Cfg.JWTTTL, userID, email, role)
	if err != nil {
		return "", "", err
	}
	plain, hash, err := auth.NewRefreshToken()
	if err != nil {
		return "", "", err
	}
	expires := time.Now().Add(time.Duration(s.Cfg.RefreshTTLDays) * 24 * time.Hour)
	if err := auth.StoreRefreshToken(ctx, s.DB, userID, hash, expires); err != nil {
		return "", "", err
	}
	return accessToken, plain, nil
}

func (s *Server) googleOAuthConfig() (*oauth2.Config, error) {
	if s.Cfg.GoogleClientID == "" || s.Cfg.GoogleClientSecret == "" || s.Cfg.GoogleRedirectURL == "" {
		return nil, errors.New("google oauth not configured")
	}
	return &oauth2.Config{
		ClientID:     s.Cfg.GoogleClientID,
		ClientSecret: s.Cfg.GoogleClientSecret,
		RedirectURL:  s.Cfg.GoogleRedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}, nil
}

func (s *Server) handleGoogleLogin(c *gin.Context) {
	cfg, err := s.googleOAuthConfig()
	if err != nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "oauth not configured"})
		return
	}
	state := "devstate" // для демо; в проде нужно подписывать/хранить per-session
	u := cfg.AuthCodeURL(state, oauth2.AccessTypeOnline)
	c.Redirect(http.StatusFound, u)
}

type googleUserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func (s *Server) handleGoogleCallback(c *gin.Context) {
	cfg, err := s.googleOAuthConfig()
	if err != nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "oauth not configured"})
		return
	}

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	tok, err := cfg.Exchange(ctx, code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code exchange failed"})
		return
	}

	client := cfg.Client(ctx, tok)
	resp, err := client.Get("https://openidconnect.googleapis.com/v1/userinfo")
	if err != nil || resp.StatusCode >= 300 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userinfo request failed"})
		return
	}
	defer func() { _ = resp.Body.Close() }()

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil || info.Email == "" || info.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid userinfo"})
		return
	}

	var userID int64
	var role string
	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx begin failed"})
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	err = tx.QueryRow(ctx, `SELECT user_id FROM oauth_accounts WHERE provider='google' AND provider_user_id=$1`, info.ID).Scan(&userID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		// create or link by email
		err = tx.QueryRow(ctx, `SELECT id, role FROM users WHERE email=$1`, info.Email).Scan(&userID, &role)
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
				return
			}
			role = "customer"
			var passengerID int
			err = tx.QueryRow(ctx, `INSERT INTO passengers(full_name, last_name, contact_number, passport) VALUES ($1, $2, $3, $4) RETURNING id`, info.Email, "", "N/A", "GOOGLE-"+info.ID).Scan(&passengerID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "passenger create failed"})
				return
			}
			err = tx.QueryRow(ctx, `INSERT INTO users(email, password_hash, role, passenger_id) VALUES ($1, NULL, $2, $3) RETURNING id`, info.Email, role, passengerID).Scan(&userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "user create failed"})
				return
			}
		}
		_, err = tx.Exec(ctx, `INSERT INTO oauth_accounts(user_id, provider, provider_user_id) VALUES ($1, 'google', $2) ON CONFLICT DO NOTHING`, userID, info.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "oauth link failed"})
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tx commit failed"})
		return
	}

	if role == "" {
		_ = s.DB.QueryRow(ctx, `SELECT role FROM users WHERE id=$1`, userID).Scan(&role)
	}
	access, refresh, err := s.issueTokens(ctx, userID, info.Email, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token sign failed"})
		return
	}

	// redirect back to frontend with tokens
	u, _ := url.Parse(s.Cfg.FrontendOrigin + "/oauth/callback")
	q := u.Query()
	q.Set("access_token", access)
	q.Set("refresh_token", refresh)
	u.RawQuery = q.Encode()
	c.Redirect(http.StatusFound, u.String())
}

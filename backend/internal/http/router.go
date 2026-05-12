package httpapi

import (
	"net/http"

	"airtickets/internal/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	Cfg  config.Config
	DB   *pgxpool.Pool
	HTTP *gin.Engine
}

func NewServer(cfg config.Config, db *pgxpool.Pool) *Server {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.FrontendOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	s := &Server{Cfg: cfg, DB: db, HTTP: r}

	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	api := r.Group("/api")
	{
		s.mountAuth(api)
		protected := api.Group("")
		protected.Use(AuthRequired(cfg))
		s.mountDomain(protected)
		s.mountReports(protected)
		s.mountAdmin(protected)
		s.mountCustomer(protected)
	}

	return s
}

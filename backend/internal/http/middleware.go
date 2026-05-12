package httpapi

import (
	"net/http"
	"strings"

	"airtickets/internal/auth"
	"airtickets/internal/config"

	"github.com/gin-gonic/gin"
)

const CtxClaimsKey = "claims"

func AuthRequired(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
		claims, err := auth.ParseJWT(cfg.JWTIssuer, cfg.JWTSecret, token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set(CtxClaimsKey, claims)
		c.Next()
	}
}

func MustClaims(c *gin.Context) *auth.Claims {
	v, _ := c.Get(CtxClaimsKey)
	claims, _ := v.(*auth.Claims)
	return claims
}

func RequireRole(roles ...string) gin.HandlerFunc {
	set := map[string]struct{}{}
	for _, r := range roles {
		set[r] = struct{}{}
	}
	return func(c *gin.Context) {
		claims := MustClaims(c)
		if claims == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if _, ok := set[claims.Role]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}

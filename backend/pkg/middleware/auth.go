package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "missing token"})
			return
		}
		// Схема обязательна: голый токен без "Bearer " не принимается.
		scheme, tokenStr, found := strings.Cut(authHeader, " ")
		if !found || scheme != "Bearer" || tokenStr == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid authorization header"})
			return
		}

		token, err := jwt.Parse(tokenStr,
			func(t *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			},
			// Явный allowlist алгоритмов вместо проверки семейства
			// и обязательный exp: токен без срока действия невалиден.
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			jwt.WithExpirationRequired(),
		)
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid claims"})
			return
		}
		if claims["type"] != "access" {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid token type"})
			return
		}
		uid, ok := claims["user_id"].(float64)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid claims"})
			return
		}
		c.Set("user_id", int64(uid))
		c.Next()
	}
}

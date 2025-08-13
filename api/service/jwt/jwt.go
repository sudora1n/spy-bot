package jwt

import (
	"fmt"
	"ssuspy-api/types"
	"strconv"
	"time"

	"connectrpc.com/authn"
	"github.com/golang-jwt/jwt/v5"
)

type JwtService struct {
	JwtSecret string
}

func NewJwtService(JwtSecret string) *JwtService {
	return &JwtService{
		JwtSecret: JwtSecret,
	}
}

func (j *JwtService) ValidateJwt(tokenValue string) (*types.User, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenValue, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return j.JwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, authn.Errorf("invalid token")
	}

	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, authn.Errorf("token expired")
	}
	if claims.Subject == "" {
		return nil, authn.Errorf("missing subject")
	}

	userId, err := strconv.ParseInt(claims.Subject, 10, 64)

	return &types.User{Id: userId}, nil
}

func (j *JwtService) IssueJwt(sub string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   sub,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(j.JwtSecret)
}

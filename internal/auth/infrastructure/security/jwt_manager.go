package security

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"efficient-request-queueing-for-llm-inference/internal/auth/domain"
	"efficient-request-queueing-for-llm-inference/internal/auth/infrastructure/config"
)

type JWTConfig struct {
	AccessSecret []byte
	Issuer       string
	Audience     string
	Subject      string
	AccessExpiry time.Duration
}

type JwtService struct {
	config JWTConfig
}

func NewJWTService(cfg config.AuthConfig) *JwtService {
	return &JwtService{
		config: JWTConfig{
			AccessSecret: []byte(cfg.JwtToken.JWTAccessSecret),
			Issuer:       cfg.JwtToken.JWTIssuer,
			Audience:     cfg.JwtToken.JWTAudience,
			Subject:      cfg.JwtToken.JWTSubject,
			AccessExpiry: cfg.JwtToken.JWTAccessExpiry,
		},
	}
}

type customClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func (s *JwtService) Generate(claims *domain.AuthenticatedUser) (string, error) {
	if claims == nil {
		return "", fmt.Errorf("claims cannot be nil")
	}

	accessToken, err := s.generateToken(claims, s.config.AccessSecret, s.config.AccessExpiry)
	if err != nil {
		return "", err
	}

	return accessToken, nil
}

func (s *JwtService) generateToken(claims *domain.AuthenticatedUser, secret []byte, expiry time.Duration) (string, error) {
	now := time.Now()

	tokenClaims := customClaims{
		UserID: claims.UserID,
		Role:   claims.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.Issuer,
			Subject:   s.config.Subject,
			Audience:  jwt.ClaimStrings{s.config.Audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expiry))),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims)
	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("jwt signing: %w", err)
	}

	return tokenString, nil
}

func (s *JwtService) ValidateAccessToken(tokenString string) (userid string, role string, err error) {
	claims, err := s.validateToken(tokenString, s.config.AccessSecret)
	if err != nil {
		return "", "", err
	}

	return claims.UserID, claims.Role, nil
}

func (s *JwtService) validateToken(tokenString string, secret []byte) (*customClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &customClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf(
				"unexpected signing method: %v",
				token.Header["alg"],
			)
		}
		return secret, nil
	},
		jwt.WithAudience(s.config.Audience),
		jwt.WithIssuer(s.config.Issuer),
	)

	if err != nil {
		return nil, fmt.Errorf("jwt parsing: %w", err)
	}

	claims, ok := token.Claims.(*customClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("jwt token valid but claims assertion failed")
	}

	if claims.Subject != s.config.Subject {
		return nil, fmt.Errorf("invalid token subject: expected 'access-token', got '%s'", claims.Subject)
	}

	if !token.Valid {
		return nil, fmt.Errorf("jwt tokenis invalid")
	}

	return claims, nil
}

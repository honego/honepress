package server

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	authCookieName      = "auth_jwt"
	accessTokenLifetime = time.Hour
)

type jwtClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Exp    int64  `json:"exp"`
}

func resolveJWTSecret() ([]byte, error) {
	if secret := strings.TrimSpace(os.Getenv("HONEPRESS_JWT_SECRET")); secret != "" {
		return []byte(secret), nil
	}
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, fmt.Errorf("read JWT secret random bytes: %w", err)
	}
	return secretBytes, nil
}

func (server *Server) setAuthCookie(responseWriter http.ResponseWriter, userID string, role string) error {
	token, err := server.signJWT(jwtClaims{
		UserID: userID,
		Role:   role,
		Exp:    time.Now().Add(accessTokenLifetime).Unix(),
	})
	if err != nil {
		return err
	}
	http.SetCookie(responseWriter, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(accessTokenLifetime.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
	})
	return nil
}

func clearAuthCookie(responseWriter http.ResponseWriter) {
	http.SetCookie(responseWriter, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
	})
}

func (server *Server) authenticatedClaims(request *http.Request) (jwtClaims, bool) {
	authCookie, err := request.Cookie(authCookieName)
	if err != nil {
		return jwtClaims{}, false
	}
	claims, err := server.verifyJWT(authCookie.Value)
	return claims, err == nil
}

func (server *Server) signJWT(claims jwtClaims) (string, error) {
	headerJSON, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	unsignedToken := encodedHeader + "." + encodedPayload
	signature := server.jwtSignature(unsignedToken)
	return unsignedToken + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (server *Server) verifyJWT(token string) (jwtClaims, error) {
	tokenParts := strings.Split(token, ".")
	if len(tokenParts) != 3 {
		return jwtClaims{}, fmt.Errorf("invalid token format")
	}
	unsignedToken := tokenParts[0] + "." + tokenParts[1]
	expectedSignature := server.jwtSignature(unsignedToken)
	actualSignature, err := base64.RawURLEncoding.DecodeString(tokenParts[2])
	if err != nil {
		return jwtClaims{}, fmt.Errorf("decode token signature: %w", err)
	}
	if subtle.ConstantTimeCompare(actualSignature, expectedSignature) != 1 {
		return jwtClaims{}, fmt.Errorf("invalid token signature")
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(tokenParts[0])
	if err != nil {
		return jwtClaims{}, fmt.Errorf("decode token header: %w", err)
	}
	var header map[string]string
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return jwtClaims{}, fmt.Errorf("decode token header JSON: %w", err)
	}
	if header["alg"] != "HS256" {
		return jwtClaims{}, fmt.Errorf("unsupported token algorithm")
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(tokenParts[1])
	if err != nil {
		return jwtClaims{}, fmt.Errorf("decode token claims: %w", err)
	}
	var claims jwtClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return jwtClaims{}, fmt.Errorf("decode token claims JSON: %w", err)
	}
	if strings.TrimSpace(claims.UserID) == "" || strings.TrimSpace(claims.Role) == "" {
		return jwtClaims{}, fmt.Errorf("token is missing identity")
	}
	if claims.Exp <= time.Now().Unix() {
		return jwtClaims{}, fmt.Errorf("token expired at %s", strconv.FormatInt(claims.Exp, 10))
	}
	return claims, nil
}

func (server *Server) jwtSignature(unsignedToken string) []byte {
	mac := hmac.New(sha256.New, server.jwtSecret)
	mac.Write([]byte(unsignedToken))
	return mac.Sum(nil)
}

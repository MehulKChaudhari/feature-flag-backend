package jwt

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtUtilsInstance *JWTUtils
	initError        error
	once             sync.Once
)

const publicKeyString = `-----BEGIN PUBLIC KEY-----
MIGeMA0GCSqGSIb3DQEBAQUAA4GMADCBiAKBgHo6sGbw8qk+XU9sBVa4w2aEq01i
oKDMFFQa9mPy0MRScTCsrfEjbypD4VqIjJcwXGmDWKVhMcJ8SMZuJumIJ10vU9ca
WSh/aHhAxiOIqOEe54IyYTwjcn5avdZry3zl62RYQ7tDZCPAR/WvFCIkgRXwjXfC
Xpm4LR6ynKDMvsDNAgMBAAE=
-----END PUBLIC KEY-----`

type JWTUtils struct {
	publicKey *rsa.PublicKey
}

func GetInstance() (*JWTUtils, error) {
	once.Do(func() {
		jwtUtilsInstance = &JWTUtils{}
		if err := jwtUtilsInstance.initialize(); err != nil {
			initError = fmt.Errorf("internal server error")
			jwtUtilsInstance = nil
		}
	})

	if initError != nil {
		return nil, initError
	}

	if jwtUtilsInstance == nil {
		return nil, errors.New("internal server error")
	}

	return jwtUtilsInstance, nil
}

func (j *JWTUtils) initialize() error {
	if j == nil {
		return errors.New("internal server error")
	}

	block, _ := pem.Decode([]byte(publicKeyString))
	if block == nil {
		return errors.New("internal server error")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("internal server error")
	}

	rsaPublicKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return errors.New("internal server error")
	}

	j.publicKey = rsaPublicKey
	return nil
}

func (j *JWTUtils) ValidateToken(tokenString string) (jwt.MapClaims, error) {
	if j == nil || j.publicKey == nil {
		return nil, errors.New("internal server error")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("invalid token")
		}
		return j.publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token")
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

func (j *JWTUtils) ExtractClaim(claims jwt.MapClaims, claimKey string) (string, error) {
	if claims == nil {
		return "", errors.New("internal server error")
	}

	value, ok := claims[claimKey].(string)
	if !ok || value == "" {
		return "", fmt.Errorf("unauthorized")
	}
	return value, nil
}

func JWTMiddleware() func(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, string, error) {
	return func(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, string, error) {
		jwtUtils, err := GetInstance()
		if err != nil {
			log.Printf("Failed to get JWTUtils instance: %v", err)
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       "Internal server error",
			}, "", nil
		}

		cookie := req.Headers["Cookie"]
		if cookie == "" {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusUnauthorized,
				Body:       "Unauthenticated",
			}, "", nil
		}

		cookieName := os.Getenv("SESSION_COOKIE_NAME")
		if cookieName == "" {
			cookieName = "rds-development-session"
		}

		var jwtToken string
		cookies := strings.Split(cookie, ";")
		for _, c := range cookies {
			c = strings.TrimSpace(c)
			if strings.HasPrefix(c, cookieName+"=") {
				jwtToken = strings.TrimPrefix(c, cookieName+"=")
				break
			}
		}

		if jwtToken == "" {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusUnauthorized,
				Body:       "Unauthenticated",
			}, "", nil
		}

		claims, err := jwtUtils.ValidateToken(jwtToken)
		if err != nil {
			log.Printf("Token validation failed: %v", err)
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusUnauthorized,
				Body:       "Invalid token",
			}, "", nil
		}

		userId, err := jwtUtils.ExtractClaim(claims, "userId")
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusUnauthorized,
				Body:       "Unauthorized",
			}, "", nil
		}

		return events.APIGatewayProxyResponse{StatusCode: http.StatusOK}, userId, nil
	}
}

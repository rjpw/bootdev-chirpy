package auth

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func GetBearerToken(headers http.Header) (string, error) {

	re := regexp.MustCompile(`^Bearer\s+([A-Za-z0-9-_=]+\.[A-Za-z0-9-_=]+\.[A-Za-z0-9-_=]+)$`)

	matches := re.FindStringSubmatch(headers.Get("Authorization"))

	if len(matches) > 1 {
		token := matches[1]
		return token, nil
	} else {
		return "", errors.New("No valid Bearer token found")
	}

}

func HashPassword(password string) (string, error) {
	hashedPass, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", err
	}
	return hashedPass, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	now := time.Now()
	expiresAt := now.Add(expiresIn)
	signSecret := []byte(tokenSecret)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	})

	signedTokenString, err := token.SignedString(signSecret)
	if err != nil {
		return "", err
	}
	return signedTokenString, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&jwt.RegisteredClaims{},
		func(token *jwt.Token) (any, error) {
			return []byte(tokenSecret), nil
		})
	if err != nil {
		return uuid.Nil, err
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		fmt.Println("Token is valid for subject:", claims.Subject)
		subjectUUID, err := uuid.Parse(claims.Subject)
		if err != nil {
			return uuid.Nil, err
		}
		return subjectUUID, nil
	} else {
		return uuid.Nil, errors.New("Invalid token")
	}
}

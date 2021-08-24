package jwt

import (
	"errors"
	"log"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var secret = []byte("never cast aside and never give up")

func CreateToken(username string, second time.Duration) (string, error) {
	claims := &jwt.StandardClaims{
		ExpiresAt: time.Now().Add(time.Second * second).Unix(),
		Issuer:    username,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secret)
	if err != nil {
		log.Fatal(err)
	}

	return tokenString, err
}

func ValidateToken(tokenString string) (bool, error) {
	if tokenString == "" {
		return false, errors.New("token is empty")
	}

	claims, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	if claims == nil {
		log.Println(err)
		return false, errors.New(err.Error())
	}

	if claims.Valid {
		return true, nil
	} else if ve, ok := err.(*jwt.ValidationError); ok {
		if ve.Errors&jwt.ValidationErrorMalformed != 0 {
			return false, errors.New("That's not even a token")
		} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
			return false, errors.New("Timing is everything")
		} else {
			return false, err
		}
	} else {
		return false, err
	}
}

func RefreshToken(tokenString string, second time.Duration) (string, error) {
	if tokenString == "" {
		return "", errors.New("token is empty")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if token == nil {
		return "", errors.New(err.Error())
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New(err.Error())
	}

	username := claims["iss"].(string)
	newTokenString, err := CreateToken(username, second)
	if err != nil {
		return "", errors.New(err.Error())
	}

	return newTokenString, nil
}

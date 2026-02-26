package jwt

import (
	"encoding/json"
	"fmt"
	types "go-boilerplate/internal/common/type"
	"go-boilerplate/internal/pkg/helper"
	"go-boilerplate/internal/pkg/logger"
	"go-boilerplate/internal/pkg/validation"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	UserDataKey = "user_data"
)

func getJWTSecret() []byte {
	secret := helper.GetEnv("JWT_SECRET")
	if secret == "" {
		logger.Warning.Println("JWT_SECRET not found, using default secret")
		secret = "$d3f4uIt_s3cr3t_key#"
	}
	return []byte(secret)
}

func GenerateToken(data types.UserWithAuth) (string, *time.Time) {
	var tokenDuration = 24 * time.Hour
	exp := time.Now().Add(tokenDuration)

	claims := jwt.MapClaims{
		"exp":       exp.Unix(),
		UserDataKey: data,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte(getJWTSecret()))
	if err != nil {
		return "", nil
	}

	return signedToken, &exp
}

func ValidateToken(jwtToken string) (*types.UserWithAuth, error) {
	token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(getJWTSecret()), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		var userData types.UserWithAuth
		// Extract user data from claims
		if claims[UserDataKey] == nil {
			return nil, fmt.Errorf("user data not found in token claims")
		}

		userDataBytes, err := json.Marshal(claims[UserDataKey])
		if err != nil {
			return nil, fmt.Errorf("error marshalling user data: %v", err)
		}

		err = json.Unmarshal(userDataBytes, &userData)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling user data: %v", err)
		}

		err = validation.Validate(userData)
		if err != nil {
			return nil, err
		}

		fmt.Println("Valid token claims", userData)

		return &userData, nil
	}

	return nil, fmt.Errorf("invalid token")
}

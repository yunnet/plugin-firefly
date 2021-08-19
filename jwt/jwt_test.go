package jwt

import (
	"fmt"
	"testing"
)

func Test_GetToken(t *testing.T) {
	tokenString, _ := CreateToken("admin", 60)
	fmt.Println("token: " + tokenString)
}

func Test_ValidateToken(t *testing.T) {
	tokenString := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MjkyODM0MjEsImlzcyI6ImFkbWluIn0.5KbO4Y4HM4SlE93JOkn63gL8U_Dc1ZQx7tN71azEoP0"
	valid, _ := ValidateToken(tokenString)
	if valid {
		fmt.Println("pass")
	} else {
		fmt.Println("no pass")
	}
}

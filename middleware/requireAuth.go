package middleware

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/khaylila/go-trendDecoration/config"
	"github.com/khaylila/go-trendDecoration/initializers"
	"github.com/khaylila/go-trendDecoration/models"
)

func Unauthorized() fiber.Map {
	return fiber.Map{
		"code":   "401",
		"status": "UNAUTHORIZED",
		"erorr": fiber.Map{
			"message": "Login terlebih dahulu untuk melanjutkan.",
		},
	}
}

func InternalServerError() fiber.Map {
	return fiber.Map{
		"code":   "500",
		"status": "INTERNAL_SERVER_ERROR",
		"erorr": fiber.Map{
			"message": "Terjadi kesalahan pada sisi server.",
		},
	}
}

func RequireAuth(c *fiber.Ctx) error {
	var header struct {
		Authorization string `reqHeader:"Authorization"`
	}

	if err := c.ReqHeaderParser(&header); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(Unauthorized())
	}

	// get the cookie
	tokenString := header.Authorization
	if tokenString == "" {
		tokenString = c.Cookies("Authorization")
	}
	if tokenString == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(Unauthorized())
	}

	if len(strings.Split(tokenString, " ")) > 1 {
		tokenString = strings.Split(tokenString, " ")[1]
	}

	if tokenString == "undefined" {
		return c.Status(fiber.StatusInternalServerError).JSON(InternalServerError())
	}

	// decode/validate
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(os.Getenv("JWT.SECRET")), nil
	})
	if err != nil {
		log.Fatal(err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		// check the exp
		if float64(time.Now().Unix()) > claims["exp"].(float64) {
			return c.Status(fiber.StatusUnauthorized).JSON(Unauthorized())
		}

		// find the user with token sub
		var user models.User

		// initializers.DB.First(&user, claims["sub"])

		result := initializers.DB.Raw("SELECT * FROM users WHERE id = ?", claims["sub"]).Scan(&user)
		if result.Error != nil {
			fmt.Println(result.Error)
			return c.Status(fiber.StatusInternalServerError).JSON(InternalServerError())
		}
		result = initializers.DB.Raw("SELECT roles.* FROM user_role JOIN roles ON role_id = roles.id WHERE user_id = ?", claims["sub"]).Scan(&user.Role)
		if result.Error != nil {
			fmt.Println(result.Error)
			return c.Status(fiber.StatusInternalServerError).JSON(InternalServerError())
		}
		if user.ID == 0 {
			return c.Status(fiber.StatusUnauthorized).JSON(Unauthorized())
		}

		// attach to req
		c.Locals("user", user)

		// continue
		return c.Next()

	}
	return c.Status(fiber.StatusUnauthorized).JSON(Unauthorized())
}

func CheckRole(reqRole string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// get user role
		user := c.Locals("user").(models.User)
		userRoles := user.Role

		roleMatch := false
		for _, role := range userRoles {
			if reqRole == role.Role {
				roleMatch = true
				break
			}
		}

		if !roleMatch {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		var merchant models.Merchant
		if reqRole == config.SELLER {
			if result := initializers.DB.Raw("SELECT * FROM merchants WHERE user_id=?", user.ID).Scan(&merchant); result.Error != nil {
				return c.SendStatus(fiber.StatusInternalServerError)
			}
			c.Locals("merchant", merchant)
		}

		return c.Next()
	}
}

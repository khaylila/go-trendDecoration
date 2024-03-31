package controllers

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/khaylila/go-trendDecoration/initializers"
	"github.com/khaylila/go-trendDecoration/models"
	"golang.org/x/crypto/bcrypt"
)

func SignUp(c *fiber.Ctx) error {
	// get the email/pass off req body
	var body struct {
		Email          string `json:"email" form:"email"`
		Password       string `json:"password" form:"password"`
		RepeatPassword string `json:"repeatPassword" form:"repeatPassword"`
		FirstName      string `json:"firstName" form:"firstName"`
		LastName       string `json:"lastName" form:"lastName"`
	}

	if c.BodyParser(&body) != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to read body.",
		})
	}

	// hash the password
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), 10)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to hash password.",
		})
	}

	// // create the user
	var role models.Role
	// if
	initializers.DB.Raw("SELECT * FROM roles WHERE id=?", 4).Scan(&role)
	if role.ID == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to query roles.",
		})
	}

	user := models.User{Email: body.Email, Password: string(hash), FirstName: body.FirstName, LastName: body.LastName, IsActive: true, Role: []models.Role{role}}
	result := initializers.DB.Create(&user)

	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create user.",
		})
	}

	// respond
	return c.Status(fiber.StatusOK).JSON(fiber.Map{}, "application/vnd.api+json")
}

func Login(c *fiber.Ctx) error {
	// get the email/pass off req body
	var body struct {
		Email          string `json:"email" form:"email"`
		Password       string `json:"password" form:"password"`
		RepeatPassword string `json:"repeatPassword" form:"repeatPassword"`
		FirstName      string `json:"firstName" form:"firstName"`
		LastName       string `json:"lastName" form:"lastName"`
	}

	if c.BodyParser(&body) != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to read body.",
		})
	}

	// look up registered user
	var user models.User
	initializers.DB.First(&user, "email = ?", body.Email)

	if user.ID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user email or password.",
		})
	}

	// compare sent in password with saved password
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid user email or password.",
		})
	}

	// generate jwt token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(time.Hour * 24 * 1).Unix(),
	})

	// sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT.SECRET")))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Unable to generate JWT token.",
		})
	}

	// create cookie
	cookie := new(fiber.Cookie)
	cookie.Name = "Authorization"
	cookie.Value = tokenString
	cookie.MaxAge = 3600 * 24 * 1
	cookie.Secure = false
	cookie.HTTPOnly = true
	cookie.Expires = time.Now().Add(24 * time.Hour * 1)
	cookie.SameSite = "lax"
	c.Cookie(cookie)

	// send it back
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		// "token" : tokenString,
	}, "application/vnd.api+json")
}

// func ResetPassword(c *fiber.Ctx) error {
// 	// get the email/pass off req body
// 	var body struct {
// 		Email string `json:"email" form:"email"`
// 	}

// 	if c.BodyParser(&body) != nil {
// 		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
// 			"error": "failed to read body.",
// 		})
// 	}

// 	// look up registered user
// 	var user models.User
// 	initializers.DB.First(&user, "email = ?", body.Email)

// 	if user.ID == 0 {
// 		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
// 			"error": "invalid user email or password.",
// 		})
// 	}
// }

func Validate(c *fiber.Ctx) error {
	// userObj := c.Locals("user")
	// if user, ok := userObj.(models.User); ok {
	// 	fmt.Println(user.ID)
	// 	fmt.Println(user.FirstName)
	// }
	// fmt.Println(c.Locals("hujan"))
	// if c.Locals("hujan") == nil {
	// 	fmt.Println("null")
	// }
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":   "Already login.",
		"user_data": c.Locals("user"),
	})
}

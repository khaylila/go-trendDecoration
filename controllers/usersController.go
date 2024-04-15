package controllers

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/khaylila/go-trendDecoration/initializers"
	"github.com/khaylila/go-trendDecoration/models"
	"github.com/khaylila/go-trendDecoration/validation"
	"golang.org/x/crypto/bcrypt"
)

func SignUp(c *fiber.Ctx) error {
	// get the email/pass off req body
	type BodyRegister struct {
		Email          string `json:"email" form:"email" validate:"required,email,min=6,max=32"`
		Password       string `json:"password" form:"password" validate:"required,min=6,max=32"`
		RepeatPassword string `json:"repeatpassword" form:"repeatpassword" validate:"required,eqfield=Password"`
		FirstName      string `json:"firstname" form:"firstname" validate:"required,min=1,max=32,alpha"`
		LastName       string `json:"lastname" form:"lastname" validate:"required,min=1,max=32,alpha"`
	}

	body := new(BodyRegister)

	if c.BodyParser(body) != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":   "400",
			"status": "BAD_REQUEST",
			"error": fiber.Map{
				"message": "failed to read body.",
			},
		})
	}

	errors := validation.ReturnValidation(body)

	// check unique email
	var resultEmail uint
	if result := initializers.DB.Raw("SELECT 1 FROM users WHERE email=?;", body.Email).Scan(&resultEmail); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to query user.",
			},
		}, "application/vnd.api+json")
	}

	if resultEmail == 1 {
		errors["Email"] = "Email sudah pernah ditambahkan sebelumnya."
	}

	if len(errors) != 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":   "400",
			"status": "BAD_REQUEST",
			"errors": errors,
		})
	}

	// hash the password
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), 10)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to hash password.",
			},
		}, "application/vnd.api+json")
	}

	// // create the user
	var role models.Role
	// if
	initializers.DB.Raw("SELECT * FROM roles WHERE id=?", 4).Scan(&role)
	if role.ID == 0 {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to query roles.",
			},
		}, "application/vnd.api+json")
	}

	user := models.User{Email: body.Email, Password: string(hash), FirstName: body.FirstName, LastName: body.LastName, IsActive: true, Role: []models.Role{role}}
	result := initializers.DB.Create(&user)

	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to create user.",
			},
		}, "application/vnd.api+json")
	}

	// respond
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"code":   "201",
		"status": "CREATED",
		"data": fiber.Map{
			"message": "Berhasil menambahkan user.",
		},
	}, "application/vnd.api+json")
}

func Login(c *fiber.Ctx) error {
	// get the email/pass off req body
	var body struct {
		Email    string `json:"email" form:"email" validate:"required,email"`
		Password string `json:"password" form:"password" validate:"required"`
		Remember bool   `json:"remember" form:"remember"`
	}

	if c.BodyParser(&body) != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":   "400",
			"status": "BAD_REQUEST",
			"error": fiber.Map{
				"message": "Pastikan jika parameter yang dikirimkan telah sesuai.",
			},
		}, "application/vnd.api+json")
	}

	errors := validation.ReturnValidation(body)
	if len(errors) != 0 {
		log.Println(errors)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":   "400",
			"status": "BAD_REQUEST",
			"errors": errors,
		})
	}

	// look up registered user
	var user models.User
	// initializers.DB.First(&user, "email = ?", body.Email)

	if result := initializers.DB.Raw("SELECT *,CONCAT(CAST(? AS TEXT), avatar) AS avatar FROM users WHERE email=?;", fmt.Sprintf("%s/img/", c.BaseURL()), body.Email).Scan(&user); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to query user.",
			},
		}, "application/vnd.api+json")
	}

	if user.ID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":   "401",
			"status": "UNAUTHORIZED",
			"error": fiber.Map{
				"message": "Email atau kata sandi salah.",
			},
		}, "application/vnd.api+json")
	}

	// compare sent in password with saved password
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":   "401",
			"status": "UNAUTHORIZED",
			"error": fiber.Map{
				"message": "Email atau kata sandi salah.",
			},
		}, "application/vnd.api+json")
	}

	var day int
	if body.Remember {
		day = 30
	} else {
		day = 1
	}

	// generate jwt token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(time.Hour * 24 * time.Duration(day)).Unix(),
	})

	// sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT.SECRET")))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "Terjadi kesalahan ketika men-generate JWT token.",
			},
		}, "application/vnd.api+json")
	}

	// get user role
	var role models.Role
	if result := initializers.DB.Raw("SELECT roles.* FROM user_role JOIN roles ON roles.id = user_role.role_id WHERE user_role.user_id=?;", user.ID).Scan(&role); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to query role.",
			},
		}, "application/vnd.api+json")
	}

	// create cookie
	cookie := new(fiber.Cookie)
	cookie.Name = "Authorization"
	cookie.Value = tokenString
	cookie.MaxAge = 3600 * 24 * day
	cookie.Secure = false
	cookie.HTTPOnly = true
	cookie.Expires = time.Now().Add(24 * time.Hour * 1)
	cookie.SameSite = "lax"
	c.Cookie(cookie)

	// send it back
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":   "200",
		"status": "OK",
		"data": fiber.Map{
			"token": tokenString,
			"role":  role.Role,
			"user": fiber.Map{
				"firstname": user.FirstName,
				"lastname":  user.LastName,
				"avatar":    user.Avatar,
			},
		},
	}, "application/vnd.api+json")
}

func CheckLogin(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": fiber.StatusOK,
		"data": fiber.Map{
			"message": "User ter-autentikasi.",
			"user":    c.Locals("user"),
		},
	})
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

func UserProfile(c *fiber.Ctx) error {
	var user struct {
		ID        uint   `json:"id"`
		FirstName string `json:"firstname"`
		LastName  string `json:"lastname"`
		Email     string `json:"email"`
		Avatar    string `json:"avatar"`
		UserRole  string `json:"role"`
	}

	if result := initializers.DB.Raw("SELECT users.id,first_name,last_name,email,CONCAT(CAST(? AS TEXT), avatar) AS avatar,roles.role as user_role FROM users JOIN user_role ON users.id = user_role.user_id JOIN roles ON roles.id = user_role.role_id WHERE users.id=?;", fmt.Sprintf("%s/img/", c.BaseURL()), c.Locals("user").(models.User).ID).Scan(&user); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":   "500",
			"status": "INTERNAL_SERVER_ERROR",
			"erorr": fiber.Map{
				"message": "failed to query user.",
			},
		}, "application/vnd.api+json")
	}
	log.Println(user)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"code":   "200",
		"status": "OK",
		"data":   user,
	}, "application/vnd.api+json")
}

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

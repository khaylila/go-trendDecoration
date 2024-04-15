package validation

import (
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/khaylila/go-trendDecoration/initializers"
)

func MaxLength(value string, length int) bool {
	return len(value) <= length
}

func MinLength(value string, length int) bool {
	return len(value) >= length
}

func RegexMatch(value string, pattern string) bool {
	regex := regexp.MustCompile(`^[a-zA-Z\s]+$`)
	return regex.MatchString(value)
}

func Match(value, matchValue string) bool {
	return value == matchValue
}

func IsUnique(value, table, field, ignoreField, ignoreValue string) bool {
	var result int8
	if ignoreField != "" {
		initializers.DB.Raw("SELECT 1 FROM ? WHERE ? = ? AND ? <> ?", table, field, value, ignoreField, ignoreValue).Scan(&result)
	} else {
		initializers.DB.Raw("SELECT 1 FROM ? WHERE ? = ?", table, field, value).Scan(&result)
	}
	if result != 1 {
		return false
	}
	return true
}

func ReturnValidation(body interface{}) map[string]string {
	// var errors []*models.ErrorResponse
	errors := make(map[string]string)

	validate := validator.New()

	if err := validate.Struct(body); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			field := strings.ToLower(err.Field())
			if err.Tag() == "required" {
				errors[field] = "Kolom " + err.Field() + " belum diisi."
			} else if err.Tag() == "email" {
				errors[field] = "Format email tidak sesuai."
			} else if err.Tag() == "min" {
				errors[field] = "Masukkan minimal " + err.Param() + " karakter."
			} else if err.Tag() == "max" {
				errors[field] = "Maksimal " + err.Param() + " karakter."
			} else if err.Tag() == "eqfield" {
				errors[field] = "Masukan tidak sesuai."
			}
		}
		return errors
	}
	return errors
}

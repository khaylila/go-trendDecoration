package validation

import (
	"regexp"

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

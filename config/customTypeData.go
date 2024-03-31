package config

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type DateRange struct {
	StartDate time.Time
	EndDate   time.Time
}

func (d *DateRange) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	// Assuming value is a string representation of the daterange
	stringValue, ok := value.(string)
	if !ok {
		return errors.New("failed to scan DateRange value")
	}

	// Parse the string to extract start and end dates
	parts := strings.Split(stringValue, ",")
	if len(parts) != 2 {
		return errors.New("invalid DateRange format")
	}

	// Trim spaces and brackets from start and end date strings
	startDateString := strings.TrimSpace(strings.Trim(parts[0], "[)"))
	endDateString := strings.TrimSpace(strings.Trim(parts[1], "[)"))

	// Parse the date strings into time.Time values
	startDate, err := time.Parse("2006-01-02", startDateString)
	if err != nil {
		return err
	}
	endDate, err := time.Parse("2006-01-02", endDateString)
	if err != nil {
		return err
	}

	// Assign parsed dates to DateRange struct
	d.StartDate = startDate
	d.EndDate = endDate

	return nil
}

// Implement the driver.Valuer interface
func (d DateRange) Value() (driver.Value, error) {
	return fmt.Sprintf("[%s, %s)", d.StartDate.Format("2006-01-02"), d.EndDate.Format("2006-01-02")), nil
}

// Implement the gorm.DataType interface
func (DateRange) GormDataType(gorm *gorm.DB, field *schema.Field) string {
	return "daterange"
}

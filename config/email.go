package config

import (
	"log"

	"gopkg.in/gomail.v2"
)

const (
	CONFIG_SMTP_HOST     = "smtp.gmail.com"
	CONFIG_SMTP_PORT     = 587
	CONFIG_SENDER_NAME   = "Pemuja Rahasia <megaroy123@gmail.com>"
	CONFIG_AUTH_EMAIL    = "megaroy123@gmail.com"
	CONFIG_AUTH_PASSWORD = "kbzcohrotlvwjyht"
)

func SendToEmail(to, subject, body string) bool {
	mailer := gomail.NewMessage()
	mailer.SetHeader("From", CONFIG_SENDER_NAME)
	mailer.SetHeader("To", to)
	mailer.SetHeader("Subject", subject)
	mailer.SetBody("text/html", body)

	dialer := gomail.NewDialer(
		CONFIG_SMTP_HOST,
		CONFIG_SMTP_PORT,
		CONFIG_AUTH_EMAIL,
		CONFIG_AUTH_PASSWORD,
	)

	if err := dialer.DialAndSend(mailer); err != nil {
		log.Fatal(err.Error())
		return false
	}

	log.Println("Mail sent!")
	return true
}

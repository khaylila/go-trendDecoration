package config

import (
	"os"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
)

func GenerateSnapURL(orderData snap.Request) (string, error) {
	var client snap.Client
	envi := midtrans.Sandbox
	if os.Getenv("MIDTRANS.ENV") == "production" {
		envi = midtrans.Production
	}
	client.New(os.Getenv("MIDTRANS.SERVERKEY"), envi)

	// 2. Initiate Snap request
	req := &orderData

	// 3. Request create Snap transaction to Midtrans
	snapResp, err := client.CreateTransaction(req)
	if err != nil {
		return "", err
	}

	return snapResp.RedirectURL, nil
}

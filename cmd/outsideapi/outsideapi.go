package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/nkypy/outsideapi"
	"github.com/plutov/paypal/v4"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	var apiBase string
	if os.Getenv("PAYPAL_API_MODE") == "live" {
		apiBase = paypal.APIBaseLive
	} else {
		apiBase = paypal.APIBaseSandBox
	}
	paypalClient, err := paypal.NewClient(os.Getenv("PAYPAL_CLIENT_ID"), os.Getenv("PAYPAL_CLIENT_SECRET"), apiBase)
	if err != nil {
		log.Fatal("Paypal client init error:", err)
	}
	paypalClient.SetLog(os.Stdout)

	facebookConfig := &oauth2.Config{
		ClientID:     os.Getenv("FACEBOOK_CLIENT_ID"),
		ClientSecret: os.Getenv("FACEBOOK_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("FACEBOOK_REDIRECT_URL"),
		Scopes:       []string{"public_profile", "email"},
		Endpoint:     facebook.Endpoint,
	}
	r := gin.Default()
	r.Use(func(ctx *gin.Context) {
		ctx.Set("paypal", paypalClient)
		ctx.Set("facebook", facebookConfig)
	})
	outsideapi.PaypalRouter(r)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r.Run(":" + port)
}

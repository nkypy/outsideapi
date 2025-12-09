package main

import (
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/plutov/paypal/v4"
	"golang.org/x/oauth2"

	"github.com/nkypy/outsideapi"
)

func main() {
	// load .env if present
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := gin.Default()

	// Init PayPal client if credentials provided
	ppClientID := os.Getenv("PAYPAL_CLIENT_ID")
	ppSecret := os.Getenv("PAYPAL_SECRET")
	if ppClientID != "" && ppSecret != "" {
		var apiBase string
		if os.Getenv("PAYPAL_API_MODE") == "live" {
			apiBase = paypal.APIBaseLive
		} else {
			apiBase = paypal.APIBaseSandBox
		}
		pp, err := paypal.NewClient(ppClientID, ppSecret, apiBase)
		if err != nil {
			log.Println("failed to create paypal client:", err)
		} else {
			if _, err := pp.GetAccessToken(context.Background()); err != nil {
				log.Println("paypal token error:", err)
			} else {
				// inject paypal client into requests
				r.Use(func(c *gin.Context) {
					c.Set("paypal", pp)
					c.Next()
				})
			}
		}
	}

	// Init Facebook oauth config if provided
	fbID := os.Getenv("FACEBOOK_CLIENT_ID")
	fbSecret := os.Getenv("FACEBOOK_CLIENT_SECRET")
	fbRedirect := os.Getenv("FACEBOOK_REDIRECT_URL")
	if fbID != "" && fbSecret != "" {
		fbConfig := &oauth2.Config{
			ClientID:     fbID,
			ClientSecret: fbSecret,
			RedirectURL:  fbRedirect,
			Scopes:       []string{"email"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://www.facebook.com/v10.0/dialog/oauth",
				TokenURL: "https://graph.facebook.com/v10.0/oauth/access_token",
			},
		}
		r.Use(func(c *gin.Context) {
			c.Set("facebook", fbConfig)
			c.Next()
		})
	}

	// Register routers
	outsideapi.PaypalRouter(r)
	outsideapi.FacebookRouter(r)

	log.Println("listening on :" + port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

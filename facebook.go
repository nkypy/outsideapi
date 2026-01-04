package outsideapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

type FacebookLoginResponse struct {
	URL   string `json:"url"`
	State string `json:"state"`
}

type FacebookCallbackRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state" binding:"required"`
}

type FacebookCallbackResponse struct {
	AccessToken string `json:"access_token"`
	Name        string `json:"name"`
	Email       string `json:"email"`
}

func generateRandomState(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func facebookLoginHandler(c *gin.Context) {
	v, ok := c.Get("facebook")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "facebook config not found"})
		return
	}
	fbConfig, ok := v.(*oauth2.Config)
	if !ok || fbConfig == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid facebook config"})
		return
	}
	state, err := generateRandomState(18)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
		return
	}
	url := fbConfig.AuthCodeURL(state)
	// Note: for a secure implementation, store `state` per-user (session or DB) and validate it on callback.
	c.JSON(http.StatusOK, FacebookLoginResponse{URL: url, State: state})
}

func facebookCallbackHandler(c *gin.Context) {
	// Support both query parameters (typical OAuth redirect) and JSON POST bodies.
	code := c.Query("code")
	state := c.Query("state")
	if code == "" {
		var params FacebookCallbackRequest
		if err := c.BindJSON(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		code = params.Code
		state = params.State
	}
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	v, ok := c.Get("facebook")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "facebook config not found"})
		return
	}
	fbConfig, ok := v.(*oauth2.Config)
	if !ok || fbConfig == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid facebook config"})
		return
	}

	// Note: validate `state` against stored value for CSRF protection.
	token, err := fbConfig.Exchange(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to exchange token: " + err.Error()})
		return
	}
	client := fbConfig.Client(context.Background(), token)
	resp, err := client.Get("https://graph.facebook.com/me?fields=name,email")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user info: " + err.Error()})
		return
	}
	defer resp.Body.Close()
	var fbUser struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&fbUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode user info: " + err.Error()})
		return
	}
	body := FacebookCallbackResponse{
		AccessToken: token.AccessToken,
		Name:        fbUser.Name,
		Email:       fbUser.Email,
	}
	bodyBytes, _ := json.Marshal(&body)
	println("facebook callback received:", string(bodyBytes))
	callbackURL := os.Getenv("FACEBOOK_CALLBACK_URL")
	if callbackURL != "" {
		go func(b []byte, url string) {
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Post(url, "application/json", bytes.NewBuffer(b))
			if err != nil {
				println("facebook callback forward error:", err.Error())
				return
			}
			defer resp.Body.Close()
		}(bodyBytes, callbackURL)
	}
	c.JSON(http.StatusOK, gin.H{"status": "callback received", "state": state})
}

func FacebookRouter(router *gin.Engine) {
	api := router.Group("/fb")
	api.GET("/login", facebookLoginHandler)
	api.GET("/callback", facebookCallbackHandler)
	api.POST("/callback", facebookCallbackHandler)
}

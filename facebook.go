package outsideapi

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

type FacebookLoginResponse struct {
	URL string `json:"url"`
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

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateRandomString(length int) string {
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

var oauthStateString = generateRandomString(16)

func facebookLoginHandler(c *gin.Context) {
	config, ok := c.Get("facebook")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "facebook config not found"})
		return
	}
	fbConfig := config.(*oauth2.Config)
	url := fbConfig.AuthCodeURL(oauthStateString)
	c.JSON(http.StatusOK, FacebookLoginResponse{URL: url})
}

func facebookCallbackHandler(c *gin.Context) {
	var params FacebookCallbackRequest
	if err := c.ShouldBind(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config, ok := c.Get("facebook")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "facebook config not found"})
		return
	}
	fbConfig := config.(*oauth2.Config)
	token, err := fbConfig.Exchange(c, params.Code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to exchange token: " + err.Error()})
		return
	}
	client := fbConfig.Client(c, token)
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
	http.Post(os.Getenv("FACEBOOK_CALLBACK_URL"), "application/json", bytes.NewBuffer(bodyBytes))
	c.JSON(http.StatusOK, gin.H{"status": "callback received"})
}

func FacebookRouter(router *gin.Engine) {
	api := router.Group("/fb")
	api.GET("/login", facebookLoginHandler)
	api.POST("/callback", facebookCallbackHandler)
}

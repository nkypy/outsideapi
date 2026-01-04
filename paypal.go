package outsideapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/plutov/paypal/v4"
)

// 第一步
// 创建订单请求和响应结构体
type PaypalCreateOrderRequest struct {
	PurchaseUnits []paypal.PurchaseUnitRequest `json:"purchase_units"`
	PaymentSource *paypal.PaymentSource        `json:"payment_source"`
	AppContext    *paypal.ApplicationContext   `json:"app_context"`
}
type PaypalCreateOrderResponse paypal.Order

// 第二步
// 用户点击 order.Links[1].Href,确认支付后,PayPal会重定向到ReturnURL并携带token参数

// 第三步
// 捕获订单请求和响应结构体
type PaypalCaptureOrderRequest paypal.CaptureOrderRequest
type PaypalCaptureOrderResponse paypal.CaptureOrderResponse

// 第四步
// 获取订单详情响应结构体
type PaypalGetOrderRequest struct{}
type PaypalGetOrderResponse paypal.Order

// 第五步
// 获取捕获详情响应结构体
type PaypalGetCaptureRequest struct{}
type PaypalGetCaptureResponse paypal.CaptureDetailsResponse

// 第六步
// 退款请求和响应结构体
type PaypalRefundCaptureRequest paypal.RefundCaptureRequest
type PaypalRefundCaptureResponse paypal.RefundResponse

func paypalCreateOrderHandler(c *gin.Context) {
	var params PaypalCreateOrderRequest
	if err := c.BindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	v, ok := c.Get("paypal")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "paypal client not found"})
		return
	}
	pp, ok := v.(*paypal.Client)
	if !ok || pp == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid paypal client"})
		return
	}
	order, err := pp.CreateOrder(c.Request.Context(), paypal.OrderIntentCapture, params.PurchaseUnits, params.PaymentSource, params.AppContext)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if order == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "empty order returned"})
		return
	}
	c.JSON(http.StatusOK, PaypalCreateOrderResponse(*order))
}

func paypalCaptureOrderHandler(c *gin.Context) {
	orderID := c.Param("id")
	var params PaypalCaptureOrderRequest
	if err := c.BindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	v, ok := c.Get("paypal")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "paypal client not found"})
		return
	}
	pp, ok := v.(*paypal.Client)
	if !ok || pp == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid paypal client"})
		return
	}
	captureResponse, err := pp.CaptureOrder(c.Request.Context(), orderID, paypal.CaptureOrderRequest(params))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if captureResponse == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "empty capture response"})
		return
	}
	c.JSON(http.StatusOK, PaypalCaptureOrderResponse(*captureResponse))
}

func paypalGetOrderHandler(c *gin.Context) {
	orderID := c.Param("id")
	v, ok := c.Get("paypal")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "paypal client not found"})
		return
	}
	pp, ok := v.(*paypal.Client)
	if !ok || pp == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid paypal client"})
		return
	}
	order, err := pp.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, PaypalGetOrderResponse(*order))
}

func paypalGetCaptureHandler(c *gin.Context) {
	captureID := c.Param("id")
	v, ok := c.Get("paypal")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "paypal client not found"})
		return
	}
	pp, ok := v.(*paypal.Client)
	if !ok || pp == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid paypal client"})
		return
	}
	capture, err := pp.CapturedDetail(c.Request.Context(), captureID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, PaypalGetCaptureResponse(*capture))
}

func paypalRefundCaptureHandler(c *gin.Context) {
	captureID := c.Param("id")
	var params PaypalRefundCaptureRequest
	if err := c.BindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	v, ok := c.Get("paypal")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "paypal client not found"})
		return
	}
	pp, ok := v.(*paypal.Client)
	if !ok || pp == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid paypal client"})
		return
	}
	refundResponse, err := pp.RefundCapture(c.Request.Context(), captureID, paypal.RefundCaptureRequest(params))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, PaypalRefundCaptureResponse(*refundResponse))
}

func paypalCallbackHandler(c *gin.Context) {
	var payload map[string]interface{}
	if err := c.BindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	bodyBytes, _ := json.Marshal(&payload)
	println("paypal callback received:", string(bodyBytes))
	callbackURL := os.Getenv("PAYPAL_CALLBACK_URL")
	if callbackURL == "" {
		c.JSON(http.StatusOK, gin.H{"status": "callback received, no forward configured"})
		return
	}
	// forward asynchronously with a timeout and log errors
	go func(b []byte, url string) {
		success := false
		client := &http.Client{Timeout: 5 * time.Second}
		for i := 0; i < 3 && !success; i++ {
			resp, err := client.Post(url, "application/json", bytes.NewBuffer(b))
			if err != nil || resp.StatusCode != http.StatusOK {
				// best-effort: don't block caller; log to stdout
				// in real apps use structured logging
				println("paypal callback forward error")
				resp.Body.Close()
				time.Sleep(time.Duration(i+1) * 3 * time.Second)
				continue
			}
			resp.Body.Close()
			success = true
		}
	}(bodyBytes, callbackURL)

	c.JSON(http.StatusOK, gin.H{"status": "callback received"})
}

func PaypalRouter(router *gin.Engine) {
	api := router.Group("/pp")
	api.POST("/orders", paypalCreateOrderHandler)
	api.POST("/orders/:id/capture", paypalCaptureOrderHandler)
	api.GET("/orders/:id", paypalGetOrderHandler)
	api.GET("/captures/:id", paypalGetCaptureHandler)
	api.POST("/captures/:id/refund", paypalRefundCaptureHandler)
	api.POST("/callback", paypalCallbackHandler)
}

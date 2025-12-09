package outsideapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"

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
	if err := c.ShouldBind(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	client, ok := c.Get("paypal")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "paypal client not found"})
		return
	}
	pp := client.(*paypal.Client)
	order, err := pp.CreateOrder(context.TODO(), paypal.OrderIntentCapture, params.PurchaseUnits, params.PaymentSource, params.AppContext)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, PaypalCreateOrderResponse(*order))
}

func paypalCaptureOrderHandler(c *gin.Context) {
	orderID := c.Param("id")
	var params PaypalCaptureOrderRequest
	if err := c.ShouldBind(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	client, ok := c.Get("paypal")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "paypal client not found"})
		return
	}
	pp := client.(*paypal.Client)
	captureResponse, err := pp.CaptureOrder(context.TODO(), orderID, paypal.CaptureOrderRequest(params))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, PaypalCaptureOrderResponse(*captureResponse))
}

func paypalGetOrderHandler(c *gin.Context) {
	orderID := c.Param("id")
	client, ok := c.Get("paypal")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "paypal client not found"})
		return
	}
	pp := client.(*paypal.Client)
	order, err := pp.GetOrder(context.TODO(), orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, PaypalGetOrderResponse(*order))
}

func paypalGetCaptureHandler(c *gin.Context) {
	captureID := c.Param("id")
	client, ok := c.Get("paypal")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "paypal client not found"})
		return
	}
	pp := client.(*paypal.Client)
	capture, err := pp.CapturedDetail(context.TODO(), captureID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, PaypalGetCaptureResponse(*capture))
}

func paypalRefundCaptureHandler(c *gin.Context) {
	captureID := c.Param("id")
	var params PaypalRefundCaptureRequest
	if err := c.ShouldBind(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	client, ok := c.Get("paypal")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "paypal client not found"})
		return
	}
	pp := client.(*paypal.Client)
	refundResponse, err := pp.RefundCapture(context.TODO(), captureID, paypal.RefundCaptureRequest(params))
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
	http.Post(os.Getenv("PAYPAL_CALLBACK_URL"), "application/json", bytes.NewBuffer(bodyBytes))
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

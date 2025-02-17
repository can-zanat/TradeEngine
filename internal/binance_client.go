package internal

import (
	"TradeEngine/configs"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const sellAmount = 0.001

var (
	placeOrderDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "binance_place_order_duration_seconds",
			Help: "Latency of placeOrder calls to Binance in seconds",
		},
		[]string{"operation"},
	)
	placeOrderSuccessCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "binance_place_order_success_total",
			Help: "Number of successful placeOrder calls",
		},
		[]string{"operation"}, // BUY veya SELL
	)

	placeOrderFailureCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "binance_place_order_failure_total",
			Help: "Number of failed placeOrder calls",
		},
		[]string{"operation"},
	)
)

type Client struct {
	apiKey    string
	secretKey string
	baseURL   string
}

func NewClient() *Client {
	config, err := configs.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	return &Client{
		apiKey:    config.Binance.APIKey,
		secretKey: config.Binance.SecretKey,
		baseURL:   config.Binance.BinanceURL,
	}
}

func (c *Client) BuyFunc(sValue string) (map[string]interface{}, error) {
	return c.placeOrder(sValue, "BUY", sellAmount)
}

func (c *Client) SellFunc(sValue string) (map[string]interface{}, error) {
	return c.placeOrder(sValue, "SELL", sellAmount)
}

func (c *Client) placeOrder(symbol, side string, quantity float64) (map[string]interface{}, error) {
	startTime := time.Now()

	timestamp := time.Now().UnixMilli()
	params := url.Values{}
	params.Add("symbol", symbol)
	params.Add("side", side)
	params.Add("type", "MARKET")
	params.Add("quantity", fmt.Sprintf("%.6f", quantity))
	params.Add("timestamp", fmt.Sprintf("%d", timestamp))

	signature := c.sign(params.Encode())
	params.Add("signature", signature)

	reqURL := fmt.Sprintf("%s/api/v3/order?%s", c.baseURL, params.Encode())
	req, err := http.NewRequest("POST", reqURL, http.NoBody)

	if err != nil {
		return nil, err
	}

	req.Header.Set("X-MBX-APIKEY", c.apiKey)
	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		placeOrderFailureCounter.WithLabelValues(side).Inc()
		return nil, fmt.Errorf("failed to place order: %s", string(body))
	}

	placeOrderSuccessCounter.WithLabelValues(side).Inc()

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	duration := time.Since(startTime).Seconds()
	placeOrderDuration.WithLabelValues(side).Observe(duration)
	log.Printf("Order placed successfully: %v", result)

	return result, nil
}

func (c *Client) sign(query string) string {
	mac := hmac.New(sha256.New, []byte(c.secretKey))
	mac.Write([]byte(query))

	return hex.EncodeToString(mac.Sum(nil))
}

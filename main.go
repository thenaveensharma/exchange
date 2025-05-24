package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/thenaveensharma/exchange/orderbook"
)

func main() {
	// Echo instance
	e := echo.New()
	ex := NewExchange()

	// Routes
	e.GET("/", handleHealthCheck)
	e.POST("/order", ex.handlePlaceOrder)
	e.GET("/book/:market", ex.handleGetBook)

	// Start server
	if err := e.Start(":3000"); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start server", "error", err)
	}

}

func handleHealthCheck(c echo.Context) error {
	slog.Info("server health check")
	return c.JSON(200, "server is alive")
}

type Market string

const (
	MarketEth Market = "ETH"
	MarketBtc Market = "BTC"
)

type Exchange struct {
	orderbooks map[Market]*orderbook.Orderbook
}

func NewExchange() *Exchange {
	orderbooks := make(map[Market]*orderbook.Orderbook)
	orderbooks[MarketEth] = orderbook.NewOrderbook()
	orderbooks[MarketBtc] = orderbook.NewOrderbook()
	return &Exchange{
		orderbooks,
	}
}

type OrderType string

const (
	MarketOrder OrderType = "MARKET"
	LimitOrder  OrderType = "LIMIT"
)

type PlaceOrderRequest struct {
	Type   OrderType `json:"type"`
	Bid    bool      `json:"bid"`
	Size   float64   `json:"size"`
	Price  float64   `json:"price"`
	Market Market    `json:"market"`
}

func (ex *Exchange) handlePlaceOrder(c echo.Context) error {
	var placeOrderRequest PlaceOrderRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&placeOrderRequest); err != nil {
		return err
	}

	market := Market(placeOrderRequest.Market)

	ob := ex.orderbooks[market]

	order := orderbook.NewOrder(placeOrderRequest.Bid, placeOrderRequest.Size)

	if placeOrderRequest.Type == LimitOrder {
		ob.PlaceLimitOrder(placeOrderRequest.Price, order)
	} else {
		ob.PlaceMarketOrder(order)
	}

	return c.JSON(200, map[string]any{
		"msg":   "order placed",
		"order": placeOrderRequest,
	})
}

type Order struct {
	Price     float64
	Size      float64
	Bid       bool
	Timestamp int64
}
type OrderbookData struct {
	Asks []*Order
	Bids []*Order
}

func (ex *Exchange) handleGetBook(c echo.Context) error {
	market := Market(c.Param("market"))

	ob, ok := ex.orderbooks[market]
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"msg": "market not found",
		})
	}

	orderbookData := OrderbookData{
		Asks: []*Order{},
		Bids: []*Order{},
	}
	for _, limit := range ob.Asks() {

		for _, order := range limit.Orders {
			o :=
				Order{
					Price:     limit.Price,
					Size:      order.Size,
					Bid:       order.Bid,
					Timestamp: order.Timestamp,
				}

			orderbookData.Asks = append(orderbookData.Asks, &o)

		}

	}
	for _, limit := range ob.Bids() {

		for _, order := range limit.Orders {
			o :=
				Order{
					Price:     limit.Price,
					Size:      order.Size,
					Bid:       order.Bid,
					Timestamp: order.Timestamp,
				}

			orderbookData.Bids = append(orderbookData.Bids, &o)

		}

	}
	return c.JSON(http.StatusOK, orderbookData)
}

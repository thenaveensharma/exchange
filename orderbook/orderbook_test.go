package orderbook

import (
	"fmt"
	"reflect"
	"testing"
)

func assert(t *testing.T, a, b any) {
	if !reflect.DeepEqual(a, b) {
		t.Errorf("%+v != %+v", a, b)
	}
}

func TestLimit(t *testing.T) {
	l := NewLimit(10_000)
	buyOrderA := NewOrder(true, 5)
	buyOrderB := NewOrder(true, 8)
	buyOrderC := NewOrder(true, 10)

	// Test initial state
	assert(t, l.Price, 10_000.0)
	assert(t, l.TotalVolume, 0.0)
	assert(t, len(l.Orders), 0)

	// Test adding orders
	l.AddOrder(buyOrderA)
	l.AddOrder(buyOrderB)
	l.AddOrder(buyOrderC)

	assert(t, l.TotalVolume, 23.0)
	assert(t, len(l.Orders), 3)
	assert(t, l.Orders[0], buyOrderA)
	assert(t, l.Orders[1], buyOrderB)
	assert(t, l.Orders[2], buyOrderC)

	// Test deleting order
	l.DeleteOrder(buyOrderB)
	assert(t, l.TotalVolume, 15.0)
	assert(t, len(l.Orders), 2)
	assert(t, l.Orders[0], buyOrderA)
	assert(t, l.Orders[1], buyOrderC)
	fmt.Println(l)
}

func TestPlaceOrder(t *testing.T) {
	ob := NewOrderbook()
	sellOrderA := NewOrder(false, 20)
	sellOrderB := NewOrder(false, 5)

	// Test initial state
	assert(t, len(ob.asks), 0)
	assert(t, ob.AskTotalVolume(), 0.0)

	// Test placing orders
	ob.PlaceLimitOrder(10_000, sellOrderA)
	ob.PlaceLimitOrder(20_000, sellOrderB)

	assert(t, len(ob.asks), 2)
	assert(t, ob.AskTotalVolume(), 25.0)
	assert(t, ob.asks[0].Price, 10_000.0)
	assert(t, ob.asks[1].Price, 20_000.0)
	assert(t, ob.asks[0].TotalVolume, 20.0)
	assert(t, ob.asks[1].TotalVolume, 5.0)

	// Verify order references
	assert(t, sellOrderA.Limit, ob.asks[0])
	assert(t, sellOrderB.Limit, ob.asks[1])
}

func TestPlaceMarketOrder(t *testing.T) {
	ob := NewOrderbook()
	sellOrder := NewOrder(false, 2.0)
	buyOrder := NewOrder(true, 1.5)

	// Test initial state
	assert(t, len(ob.asks), 0)
	assert(t, ob.AskTotalVolume(), 0.0)

	// Test placing limit order
	ob.PlaceLimitOrder(120, sellOrder)
	assert(t, len(ob.asks), 1)
	assert(t, ob.AskTotalVolume(), 2.0)
	assert(t, ob.asks[0].Price, 120.0)
	assert(t, sellOrder.Limit, ob.asks[0])

	// Test placing market order
	matches := ob.PlaceMarketOrder(buyOrder)
	assert(t, len(matches), 1)
	assert(t, len(ob.asks), 1)
	assert(t, ob.AskTotalVolume(), 0.5)
	assert(t, sellOrder.Size, 0.5)
	assert(t, buyOrder.Size, 0.0)

	// Verify match details
	assert(t, matches[0].Price, 120.0)
	assert(t, matches[0].SizeFilled, 1.5)
	assert(t, matches[0].Ask, sellOrder)
	assert(t, matches[0].Bid, buyOrder)

	// Verify remaining order state
	assert(t, sellOrder.IsFilled(), false)
	assert(t, buyOrder.IsFilled(), true)
}
func TestPlaceMarketOrderMultiFill(t *testing.T) {
	ob := NewOrderbook()

	// Create multiple sell orders at different price levels
	sellOrderA := NewOrder(false, 2.0) // 2.0 units at 100
	sellOrderB := NewOrder(false, 3.0) // 3.0 units at 110
	sellOrderC := NewOrder(false, 1.0) // 1.0 units at 120

	// Place the limit orders
	ob.PlaceLimitOrder(100, sellOrderA)
	ob.PlaceLimitOrder(100, sellOrderB)
	ob.PlaceLimitOrder(120, sellOrderC)

	// Verify initial state
	assert(t, len(ob.asks), 3)
	assert(t, ob.AskTotalVolume(), 6.0)
	assert(t, ob.asks[0].Price, 100.0)
	assert(t, ob.asks[1].Price, 100.0)
	assert(t, ob.asks[2].Price, 120.0)

	// Create a buy market order that will be filled by multiple sell orders
	buyOrder := NewOrder(true, 5.5) // Total buy order size is 5.5 units

	// Place the market order
	matches := ob.PlaceMarketOrder(buyOrder)

	fmt.Printf("%+v", matches)

	// Verify matches
	assert(t, len(matches), 3)
	assert(t, matches[0].Price, 100.0) // First match at lowest price
	assert(t, matches[1].Price, 100.0) // Second match at middle price

	// Verify match sizes
	assert(t, matches[0].SizeFilled, 2.0) // First order fully filled (2.0 units at 100)
	assert(t, matches[1].SizeFilled, 3.0) // Second order fully filled (3.0 units at 110)

	// Verify remaining volumes
	assert(t, sellOrderA.Size, 0.0) // First order fully filled
	assert(t, sellOrderB.Size, 0.0) // Second order fully filled
	assert(t, sellOrderC.Size, 0.5) // Third order partially filled (0.5 units remaining)
	assert(t, buyOrder.Size, 0.0)   // Buy order fully filled

	// Verify orderbook state
	assert(t, ob.AskTotalVolume(), 0.5)    // Only 0.5 units remaining in sellOrderC
	assert(t, len(ob.asks), 3)             // All price levels should still exist
	assert(t, ob.asks[2].TotalVolume, 0.5) // Only highest price level has remaining volume
}

func CancelOrder(t *testing.T) {
	ob := NewOrderbook()
	buyOrder := NewOrder(true, 4)

	ob.PlaceLimitOrder(2000, buyOrder)
	assert(t, len(ob.bids), 1)
	assert(t, ob.bids[0].Price, 2000)
	assert(t, ob.BidTotalVolume(), 4)
	ob.CancelOrder(buyOrder)
	assert(t, len(ob.bids), 0)

}

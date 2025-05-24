package orderbook

import (
	"fmt"
	"sort"
	"time"
)

type Match struct {
	Ask        *Order
	Bid        *Order
	SizeFilled float64
	Price      float64
}

type Order struct {
	Size      float64 `json:"size"`
	Bid       bool    `json:"bid"`
	Limit     *Limit  `json:"limit"`
	Timestamp int64   `json:"timestamp"`
}

func (o *Order) String() string {
	return fmt.Sprintf("[size: %.2f]", o.Size)
}

func (o *Order) IsFilled() bool {
	return o.Size == 0.0
}

func NewOrder(bid bool, size float64) *Order {
	return &Order{
		Size:      size,
		Bid:       bid,
		Timestamp: time.Now().UnixNano(),
	}
}

type Orders []*Order

func (o Orders) Len() int {
	return len(o)
}
func (o Orders) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

func (o Orders) Less(i, j int) bool {
	return o[i].Timestamp < o[j].Timestamp
}

type Limit struct {
	Price       float64
	Orders      Orders
	TotalVolume float64
}

func (l *Limit) String() string {
	return fmt.Sprintf("[price: %.2f | volume: %.2f]", l.Price, l.TotalVolume)
}
func (l *Limit) AddOrder(o *Order) {
	o.Limit = l
	l.Orders = append(l.Orders, o)
	l.TotalVolume += o.Size
}

func (l *Limit) DeleteOrder(o *Order) {
	for index, order := range l.Orders {
		if order == o {
			l.Orders[index] = l.Orders[len(l.Orders)-1]
			l.Orders = l.Orders[:len(l.Orders)-1]
			break
		}
	}
	o.Limit = nil
	l.TotalVolume -= o.Size

	sort.Sort(l.Orders)
}

func (l *Limit) Fill(o *Order) []Match {
	var (
		matches        []Match
		ordersToDelete []*Order
	)
	for _, order := range l.Orders {

		match := l.FillOrder(order, o)
		l.TotalVolume -= match.SizeFilled
		matches = append(matches, match)
		if order.IsFilled() {
			ordersToDelete = append(ordersToDelete, order)
		}
		if o.IsFilled() {
			break
		}
	}

	for _, orderToDelete := range ordersToDelete {
		l.DeleteOrder(orderToDelete)
	}
	return matches
}

func (l *Limit) FillOrder(existingOrder, newOrder *Order) Match {
	var (
		bid        *Order
		ask        *Order
		sizeFilled float64
	)

	if newOrder.Bid {
		bid = newOrder
		ask = existingOrder
	} else {
		bid = existingOrder
		ask = newOrder
	}

	if existingOrder.Size >= newOrder.Size {
		existingOrder.Size -= newOrder.Size
		sizeFilled = newOrder.Size
		newOrder.Size = 0.0
	} else {
		newOrder.Size -= existingOrder.Size
		sizeFilled = existingOrder.Size
		existingOrder.Size = 0.0
	}
	return Match{Ask: ask, Bid: bid, SizeFilled: sizeFilled, Price: l.Price}
}

type Limits []*Limit

type ByBestAsk struct{ Limits }

func (a ByBestAsk) Len() int {
	return len(a.Limits)
}
func (a ByBestAsk) Swap(i, j int) {
	a.Limits[i], a.Limits[j] = a.Limits[j], a.Limits[i]
}

func (a ByBestAsk) Less(i, j int) bool {
	return a.Limits[i].Price < a.Limits[j].Price
}

type ByBestBid struct{ Limits }

func (a ByBestBid) Len() int {
	return len(a.Limits)
}
func (a ByBestBid) Swap(i, j int) {
	a.Limits[i], a.Limits[j] = a.Limits[j], a.Limits[i]
}

func (a ByBestBid) Less(i, j int) bool {
	return a.Limits[i].Price > a.Limits[j].Price
}

func NewLimit(price float64) *Limit {
	return &Limit{
		Price:  price,
		Orders: []*Order{},
	}
}

type Orderbook struct {
	asks      []*Limit
	bids      []*Limit
	AskLimits map[float64]*Limit
	BidLimits map[float64]*Limit
}

func NewOrderbook() *Orderbook {
	return &Orderbook{
		bids:      []*Limit{},
		asks:      []*Limit{},
		AskLimits: make(map[float64]*Limit),
		BidLimits: make(map[float64]*Limit),
	}
}
func (ob *Orderbook) PlaceMarketOrder(o *Order) []Match {
	matches := []Match{}

	if o.Bid {
		if o.Size > ob.AskTotalVolume() {
			panic(fmt.Errorf("not enough volume [size: %.2f] for market order [size: %.2f]", ob.AskTotalVolume(), o.Size))
		}
		for _, limit := range ob.Asks() {

			limitMatches := limit.Fill(o)
			matches = append(matches, limitMatches...)
			if len(limit.Orders) == 0 {
				ob.clearLimit(true, limit)
			}
		}

	} else {
		if o.Size > ob.BidTotalVolume() {
			panic(fmt.Errorf("not enough volume [size: %.2f] for market order [size: %.2f]", ob.AskTotalVolume(), o.Size))
		}
		for _, limit := range ob.Bids() {

			limitMatches := limit.Fill(o)
			matches = append(matches, limitMatches...)
			if len(limit.Orders) == 0 {
				ob.clearLimit(false, limit)
			}
		}
	}

	return matches
}

func (ob *Orderbook) CancelOrder(o *Order) {
	limit := o.Limit
	limit.DeleteOrder(o)
}
func (ob *Orderbook) BidTotalVolume() float64 {
	total := 0.0
	for _, bid := range ob.bids {
		total += bid.TotalVolume
	}
	return total
}
func (ob *Orderbook) AskTotalVolume() float64 {
	total := 0.0
	for _, ask := range ob.asks {
		total += ask.TotalVolume
	}
	return total
}

func (ob *Orderbook) PlaceLimitOrder(price float64, o *Order) {

	if o.Bid {
		for _, limit := range ob.Asks() {
			if limit.Price > price {
				break
			}

			limit.Fill(o)
			if len(limit.Orders) == 0 {
				ob.clearLimit(false, limit)
			}
			if o.IsFilled() {
				return
			}
		}
	} else {
		for _, limit := range ob.Bids() {
			if limit.Price < price {
				break
			}

			limit.Fill(o)
			if len(limit.Orders) == 0 {
				ob.clearLimit(true, limit)
			}
			if o.IsFilled() {
				return
			}
		}
	}

	// If the order is not fully filled, add it to the orderbook
	if !o.IsFilled() {
		var limit *Limit
		if o.Bid {
			limit = ob.BidLimits[price]
		} else {
			limit = ob.AskLimits[price]
		}

		if limit == nil {
			limit = NewLimit(price)
			if o.Bid {
				ob.bids = append(ob.bids, limit)
				ob.BidLimits[price] = limit
			} else {
				ob.asks = append(ob.asks, limit)
				ob.AskLimits[price] = limit
			}
		}
		limit.AddOrder(o)
	}

}

func (ob *Orderbook) Asks() []*Limit {
	sort.Sort(ByBestAsk{ob.asks})
	return ob.asks
}
func (ob *Orderbook) Bids() []*Limit {
	sort.Sort(ByBestBid{ob.bids})
	return ob.bids
}

func (ob *Orderbook) clearLimit(bid bool, l *Limit) {

	if bid {
		delete(ob.BidLimits, l.Price)

		for index, limit := range ob.bids {
			if limit == l {
				ob.bids[index] = ob.bids[len(ob.bids)-1]
				ob.bids = ob.bids[:len(ob.bids)-1]
				break
			}
		}
	} else {
		delete(ob.AskLimits, l.Price)

		for index, limit := range ob.asks {
			if limit == l {
				ob.asks[index] = ob.asks[len(ob.asks)-1]
				ob.asks = ob.asks[:len(ob.asks)-1]
				break
			}
		}
	}
}

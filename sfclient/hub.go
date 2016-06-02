package sfclient

import (
	"sync"
)

type StockHub struct {
	stock   Symbol
	venue   Venue
	account string
	tl      *TickListener
	fl      *FillListener
	ticker  <-chan *TickMessage
	fills   <-chan *FillMessage
	client  *Client

	tickMu        *sync.Mutex
	tickListeners []chan *TickMessage

	fillMu        *sync.Mutex
	fillListeners []chan *FillMessage
}

type Registerer interface {
	Register(*StockHub)
}

func NewStockHub(c *Client, account string, venue Venue, stock Symbol) (*StockHub, error) {
	ret := &StockHub{stock: stock, venue: venue, account: account, client: c, tickMu: &sync.Mutex{}, fillMu: &sync.Mutex{}}

	tl, err := c.StockTicker(account, venue, stock)
	if err != nil {
		return nil, err
	}

	fl, err := c.StockFills(account, venue, stock)
	if err != nil {
		return nil, err
	}

	ticker, err := tl.Listen()
	if err != nil {
		return nil, err
	}

	fills, err := fl.Listen()
	if err != nil {
		return nil, err
	}

	ret.tl = tl
	ret.fl = fl
	ret.fills = fills
	ret.ticker = ticker

	go ret.startSendTicks()
	go ret.startSendFills()

	return ret, nil
}

func (h *StockHub) Buy(price, qty int, typ OrderType) (*OrderResponse, error) {
	return h.client.BuyOrder(h.account, h.venue, h.stock, price, qty, typ)
}

func (h *StockHub) BuyLimit(price int, qty int) (*OrderResponse, error) {
	return h.Buy(price, qty, TypeLimit)
}

// TODO: do we need price for market, I doubt it
func (h *StockHub) BuyMarket(price int, qty int) (*OrderResponse, error) {
	return h.Buy(price, qty, TypeMarket)
}
func (h *StockHub) BuyFOK(price int, qty int) (*OrderResponse, error) {
	return h.Buy(price, qty, TypeFillOrKill)
}
func (h *StockHub) BuyIOC(price int, qty int) (*OrderResponse, error) {
	return h.Buy(price, qty, TypeImmediateOrCancel)
}

func (h *StockHub) Sell(price, qty int, typ OrderType) (*OrderResponse, error) {
	return h.client.SellOrder(h.account, h.venue, h.stock, price, qty, typ)
}

func (h *StockHub) SellLimit(price, qty int) (*OrderResponse, error) {
	return h.Sell(price, qty, TypeLimit)
}

// TODO: do we need price for market, I doubt it
func (h *StockHub) SellMarket(price, qty int) (*OrderResponse, error) {
	return h.Sell(price, qty, TypeMarket)
}

func (h *StockHub) SellFOK(price, qty int) (*OrderResponse, error) {
	return h.Sell(price, qty, TypeFillOrKill)
}

func (h *StockHub) SellIOC(price, qty int) (*OrderResponse, error) {
	return h.Sell(price, qty, TypeImmediateOrCancel)
}

func (h *StockHub) RegisterComponenets(cmpts ...Registerer) {
	for _, cmpt := range cmpts {
		cmpt.Register(h)
	}
}

func (h *StockHub) RegisterToTick(recv chan *TickMessage) {
	h.tickMu.Lock()
	h.tickListeners = append(h.tickListeners, recv)
	h.tickMu.Unlock()
}

func (h *StockHub) RegisterToFills(recv chan *FillMessage) {
	h.fillMu.Lock()
	h.fillListeners = append(h.fillListeners, recv)
	h.fillMu.Lock()
}

// No way to cancel at the moment lolol
func (h *StockHub) startSendTicks() {
	for {
		msg := <-h.ticker
		h.tickMu.Lock()
		for _, ch := range h.tickListeners {
			// Non-blocking send on channels
			select {
			case ch <- msg:
			default:
			}
		}
		h.tickMu.Unlock()
	}
}

func (h *StockHub) startSendFills() {
	for {
		msg := <-h.fills
		h.fillMu.Lock()
		for _, ch := range h.fillListeners {
			// Non-blocking send on channels
			select {
			case ch <- msg:
			default:
			}
		}
		h.fillMu.Unlock()
	}
}

type elem struct {
	ask     int
	askSize int
	bid     int
	bidSize int
}

type BidAskHistory struct {
	n     int
	i     int
	mu    *sync.Mutex
	elems []elem
	ch    chan *TickMessage
}

func NewBidAskHistory(maxLimit int) *BidAskHistory {
	return &BidAskHistory{
		n:     maxLimit,
		ch:    make(chan *TickMessage),
		mu:    &sync.Mutex{},
		elems: make([]elem, maxLimit, maxLimit),
	}

}

func (h *BidAskHistory) Register(hub *StockHub) {
	hub.RegisterToTick(h.ch)
	go h.init()
}

func (h *BidAskHistory) init() {
	for {
		msg := <-h.ch
		h.mu.Lock()
		h.i++
		// Quote previous price, if currently no bid/asks
		if msg.Quote.Ask == 0 {
			h.elems[h.i%h.n].ask = h.elems[(h.i-1)%h.n].ask
		} else {
			h.elems[h.i%h.n].ask = msg.Quote.Ask
		}
		h.elems[h.i%h.n].askSize = msg.Quote.AskSize
		if msg.Quote.Bid == 0 {
			h.elems[h.i%h.n].bid = h.elems[(h.i-1)%h.n].bid
		} else {
			h.elems[h.i%h.n].bid = msg.Quote.Bid
		}
		h.elems[h.i%h.n].bidSize = msg.Quote.BidSize
		h.mu.Unlock()
	}
}

func (h *BidAskHistory) Current() (ask, askSize, bid, bidSize int) {
	h.mu.Lock()
	curr := h.elems[h.i%h.n]
	h.mu.Unlock()
	return curr.ask, curr.askSize, curr.bid, curr.bidSize
}

func (h *BidAskHistory) Avg(x int) (ask, bid int) {
	if x > h.n {
		x = h.n
	}

	var totalAsk int
	var totalBid int
	h.mu.Lock()
	curr := h.i % h.n
	for j := 0; j < x; j++ {
		e := h.elems[((curr+h.n)-j)%h.n]
		totalAsk += e.ask
		totalBid += e.bid
	}
	h.mu.Unlock()
	return totalAsk / x, totalBid / x
}

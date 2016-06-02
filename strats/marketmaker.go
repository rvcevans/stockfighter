package strats

import (
	"log"
	"sync"

	"github.com/ifross89/stockfighter/sfclient"
)

type orderInFlight struct {
	id   int
	resp chan struct{}
}

type simpleMarketMaker struct {
	account string
	venue   sfclient.Venue
	stock   sfclient.Symbol
	fillch  <-chan *sfclient.FillMessage
	tickch  <-chan *sfclient.TickMessage
	client  *sfclient.Client

	mu              *sync.RWMutex
	latestBid       *sfclient.AskBid
	latestAsk       *sfclient.AskBid
	maxExposure     int
	currentExposure int

	inFlight chan orderInFlight
}

func New(apiKey string, account string, venue sfclient.Venue, stock sfclient.Symbol, maxExposure int) (*simpleMarketMaker, error) {
	c := sfclient.New(apiKey)

	sl, err := c.StockTicker(account, venue, stock)
	if err != nil {
		return nil, err
	}

	tick, err := sl.Listen()

	fl, err := c.StockFills(account, venue, stock)
	if err != nil {
		return nil, err
	}

	fills, err := fl.Listen()
	if err != nil {
		return nil, err
	}

	return &simpleMarketMaker{fillch: fills, tickch: tick, maxExposure: maxExposure, client: c, account: account, venue: venue, stock: stock}, nil
}

// calculate current spread to use
func (mm *simpleMarketMaker) currentSpread() (ask *sfclient.AskBid, bid *sfclient.AskBid) {
	const risk = 5             // lower is more risky
	const defaultSpreadPc = 75 // in percent

	ask, bid = &sfclient.AskBid{}, &sfclient.AskBid{IsBuy: true}
	mm.mu.RLock()
	ask.Price, ask.Quantity = mm.latestAsk.Price, mm.latestAsk.Quantity
	bid.Price, bid.Quantity = mm.latestBid.Price, mm.latestBid.Quantity
	exposure := mm.currentExposure
	mm.mu.RUnlock()

	mid := (ask.Price + bid.Price) / 2
	halfSpread := (ask.Price - bid.Price) / 2

	ask.Price = mid - ((halfSpread * defaultSpreadPc) / 100)
	bid.Price = mid + ((halfSpread * defaultSpreadPc) / 100)

	// Only risk 1/5 of the way to the limit
	bid.Quantity = (mm.maxExposure - exposure) / risk
	ask.Quantity(mm.maxExposure+exposure) / risk
	return ask, bid
}

func (mm *simpleMarketMaker) execute() {
	toAsk, toBid := mm.currentSpread()

	wg := &sync.WaitGroup{}
	waitChan := make(chan struct{})
	var bidID int
	var askID int

	wg.Add(2)
	go func() {
		defer wg.Done()
		bidResp, err := mm.client.BuyOrder(mm.account, mm.venue, mm.stock, toBid.Price, toBid.Quantity, sfclient.TypeLimit)
		if err != nil {
			log.Printf("error executing bid: %v", err)
			return
		} else if !bidResp.OK {
			log.Printf("server error from bid: %s", bidResp.Error)
			return
		}
		bidID = bidResp.ID
		mm.inFlight <- orderInFlight{id: bidID, resp: waitChan}
	}()

	go func() {
		defer wg.Done()
		askResp, err := mm.client.SellOrder(mm.account, mm.venue, mm.stock, toAsk.Price, toAsk.Quantity, sfclient.TypeLimit)
		if err != nil {
			log.Printf("error executing bid: %v", err)
			return
		} else if !askResp.OK {
			log.Printf("server error from bid: %s", askResp.Error)
			return
		}
		askID = askResp.ID
		mm.inFlight <- orderInFlight{id: askID, resp: waitChan}
	}()
	wg.Wait()
	// send id 0 to let the listener know that we aren't waiting on any more
	mm.inFlight <- orderInFlight{id: 0, resp: waitChan}

	// when the listener has received a fill, cancel the others
	<-waitChan

	// Now cancel the orders
	go func() {
		r, err := mm.client.CancelOrder(mm.venue, mm.stock, bidID)
		if err != nil {
			log.Printf("error canceling bid: %v", err)
		} else if !r.OK {
			log.Printf("error cancelling bid: %v", err)
		} else {
			log.Println("bid %d successfully cancelled")
		}
	}()

	go func() {
		r, err := mm.client.CancelOrder(mm.venue, mm.stock, askID)
		if err != nil {
			log.Printf("error canceling ask: %v", err)
		} else if !r.OK {
			log.Printf("error cancelling ask: %v", err)
		} else {
			log.Println("ask %d successfully cancelled")
		}
	}()
}

type receivedFill struct {
}

func (mm *simpleMarketMaker) listen() {
	idsInFlight := make(map[int]struct{})
	fillsReceived := make(map[int]struct{})

	for {
		select {
		case msg := <-mm.tickch:
			if msg.OK {
				mm.mu.Lock()
				// update latest bid / ask prices
				if msg.Quote.Ask > 0 {
					mm.latestAsk.Price = msg.Quote.Ask
				}
				mm.latestAsk.Quantity = msg.Quote.AskSize
				if msg.Quote.Bid > 0 {
					mm.latestBid.Price = msg.Quote.Bid
				}
				mm.latestBid.Quantity = msg.Quote.BidSize
				mm.mu.Unlock()
			}
		case fill := <-mm.fillch:
			if fill.OK {
				fill.Account
			}
		}
	}
}

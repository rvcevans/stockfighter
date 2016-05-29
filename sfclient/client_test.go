package sfclient

import (
	"flag"
	"fmt"
	"testing"
	"time"
)

var (
	apiKey = flag.String("apikey", "", "specify your API key")
	c *sfclient
)

func init() {
	flag.Parse()
	c = New(*apiKey)
}

func checkerr(resp APIResponse, err error) error {
	if err != nil {
		return fmt.Errorf("error calling API: %v", err)
	}

	if !resp.OK {
		return fmt.Errorf("error from server: %s", resp.Error)
	}

	return nil
}

func TestHeartbeat(t *testing.T) {
	hr, err := c.Heartbeat()

	err = checkerr(hr.APIResponse, err)
	if err != nil {
		t.Errorf("heartbeat error: %v", err)
	}
}

func TestVenueHeartbeat(t *testing.T) {
	hr, err := c.VenueHeartbeat(testVenue)

	err = checkerr(hr.APIResponse, err)
	if err != nil {
		t.Errorf("venue heartbeat error: %v", err)
	}
}

func TestVenueStocks(t *testing.T) {
	sr, err := c.VenueStocks(testVenue)

	if err = checkerr(sr.APIResponse, err); err != nil {
		t.Errorf("error getting venue's stocks: %v", err)
		return
	}

	if len(sr.Symbols) != 1 {
		t.Errorf("expected 1 stock, got %d", len(sr.Symbols))
		return
	}

	if sr.Symbols[0].Symbol != testSymbol {
		t.Errorf("expected symbol: %s, got %s", testSymbol, sr.Symbols[0].Symbol)
	}
}

func TestStockOrderBook(t *testing.T) {
	book, err := c.StockOrderBook(testVenue, testSymbol)
	err = checkerr(book.APIResponse, err)
	if err != nil {
		t.Errorf("error getting stock book: %v", err)
		return
	}

	if book.Symbol != testSymbol {
		t.Errorf("expected symbol %s, got %s", testSymbol, book.Symbol)
		return
	}

	zeroTime := time.Time{}
	if book.Timestamp == zeroTime {
		t.Error("timestamp from server is zero timetime")
		return
	}

	t.Logf("order book response: %+v", book)
}

func TestBuyOrder(t *testing.T) {
	br, err := c.BuyOrder(testAccount, testVenue, testSymbol, 10, 10, TypeMarket)
	if err = checkerr(br.APIResponse, err); err != nil {
		t.Errorf("error sending buy request: %v", err)
		return
	}

	t.Logf("buy order response: %+v", br)
}

func TestSellOrder(t *testing.T) {
	sr, err := c.SellOrder(testAccount, testVenue, testSymbol, 10, 10, TypeMarket)
	if err = checkerr(sr.APIResponse, err); err != nil {
		t.Errorf("error sending sell order: %v", err)
		return
	}

	t.Logf("sell order response: %+v", sr)
}

func TestCancelOrder(t *testing.T) {
	// First specify a sell at a huge price so should be able to cancel
	const nstocks = 10
	const price = 1000000

	sr, err := c.SellOrder(testAccount, testVenue, testSymbol, price, nstocks, TypeLimit)
	if err = checkerr(sr.APIResponse, err); err != nil {
		t.Errorf("error placing sell order: %v", err)
		return
	}

	if !sr.Open {
		t.Error("socks got sold, whoops")
		return
	}

	cr, err := c.CancelOrder(testVenue, testSymbol, sr.ID)
	if err != nil {
		t.Errorf("error cancelling order: %v", err)
	}

	if cr.Quantity != 0 {
		t.Errorf("stocks not cancelled, still got a qty: %d", cr.Quantity)
		return
	}
}

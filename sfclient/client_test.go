package sfclient

import (
	"flag"
	"fmt"
	"testing"
	"time"
)

var (
	apiKey = flag.String("apikey", "", "specify your API key")
	c      *Client
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

// First specify a sell at a huge price so should be able to cancel
const nstocks = 10
const price = 1000000

func TestCancelOrder(t *testing.T) {
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

	if cr.Quantity != 0 {
		t.Errorf("stocks not cancelled, still got a qty: %d", cr.Quantity)
		return
	}

	t.Logf("cancel order response: %+v", cr)
}

func testOrderStatus(t *testing.T, apiFunc func() (*MultiStatusResponse, error)) {
	// execute a sell order and then gets the status of the venue
	sr, err := c.SellOrder(testAccount, testVenue, testSymbol, price, nstocks, TypeLimit)
	if err = checkerr(sr.APIResponse, err); err != nil {
		t.Errorf("error placing sell order: %v", err)
		return
	}

	if !sr.Open {
		t.Error("stocks got sold, whoops")
		return
	}

	vr, err := apiFunc()
	if err = checkerr(vr.APIResponse, err); err != nil {
		t.Errorf("error getting order status: %v", err)
		return
	}

	var hasOrder bool
	for _, order := range vr.Orders {
		if sr.ID == order.ID {
			hasOrder = true
		}
	}

	if !hasOrder {
		t.Errorf("did not get sell order in venue order status")
	}

	t.Logf("venue order status: %+v", vr)
}

func TestVenueOrdersStatus(t *testing.T) {
	testOrderStatus(t, func() (*MultiStatusResponse, error) {
		return c.VenueOrdersStatus(testAccount, testVenue)
	})
}

func TestStockOrdersStatus(t *testing.T) {
	testOrderStatus(t, func() (*MultiStatusResponse, error) {
		return c.StockOrdersStatus(testAccount, testVenue, testSymbol)
	})
}

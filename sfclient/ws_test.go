package sfclient

import (
	"fmt"
	"testing"
)

const (
	testAccount        = "EXB123456"
	testVenue   Venue  = "TESTEX"
	testSymbol  Symbol = "FOOBAR"
)

func TestConnectVenueTicker(t *testing.T) {
	ticker, err := c.VenueTicker(testAccount, testVenue)
	if err != nil {
		t.Errorf("error creating venue ticker: %v", err)
		return
	}

	defer ticker.Close()

	listener, err := ticker.Listen()
	if err != nil {
		t.Errorf("unable to connect to venue listener: %v", err)
		return
	}

	// Ensure there are at least 5 orders on to get
	for i := 0; i < 5; i++ {
		price := i + 1
		br, err := c.BuyOrder(testAccount, testVenue, testSymbol, price, 10, TypeMarket)
		if err = checkerr(br.APIResponse, err); err != nil {
			fmt.Println("error placing buy order: %v", err)
			return
		}
	}

	for i := 0; i < 5; i++ {
		msg := <-listener
		if !msg.OK {
			t.Errorf("error in message from server: %s", msg.Error)
			return
		}

		t.Logf("got message for symbol: %s", msg.Quote.Symbol)
	}
}

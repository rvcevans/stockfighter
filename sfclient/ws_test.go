package sfclient

import (
	"testing"
)

const (
	testAccount        = "EXB123456"
	testVenue   Venue  = "TESTEX"
	testSymbol  Symbol = "FOOBAR"
)

func TestConnectVenueTicker(t *testing.T) {
	t.Skip()
	c := New("")
	ticker, err := c.VenueFills(testAccount, testVenue)
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

	for i := 0; i < 5; i++ {
		msg := <-listener
		if !msg.OK {
			t.Errorf("error in message from server: %s", msg.Error)
			return
		}

		t.Logf("got message for symbol: %s", msg.Symbol)
	}
}

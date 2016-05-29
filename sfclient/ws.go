package sfclient

import (
	"github.com/gorilla/websocket"
	"log"
	"net/url"
	"path"
	"time"
)

type TickMessage struct {
	APIResponse
	Quote StockState `json:"quote"`
}

type TickListener struct {
	url      *url.URL
	close    chan struct{}
	messages chan *TickMessage
}

func (t *TickListener) Listen() (<-chan *TickMessage, error) {
	log.Println("connecting to", t.url.String())
	c, _, err := websocket.DefaultDialer.Dial(t.url.String(), nil)
	log.Println("connected to", t.url.String())
	if err != nil {
		return nil, err
	}

	go func() {
		defer close(t.messages)
		for {
			select {
			case <-t.close:
				// user specified close
				c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				c.Close()
				return
			default:
				msg := &TickMessage{}
				log.Println("trying to read msg")
				err := c.ReadJSON(msg)
				log.Println("got msg with err", err)
				if err != nil {
					// Probably a connection error. Attempt reconnect
					log.Printf("websocket read err: %v", err)
					c.Close()
					c, _, err = websocket.DefaultDialer.Dial(t.url.String(), nil)
					if err != nil {
						log.Printf("websocket reconnect fail: %v", err)
						return
					}
					// Reconnect success
				} else {
					// Send message
					t.messages <- msg
				}
			}
		}
	}()

	return t.messages, nil
}

func (t *TickListener) Close() {
	t.close <- struct{}{}
}

func (c *sfclient) VenueTicker(account string, venue Venue) (*TickListener, error) {
	u, err := url.Parse(c.baseWSURL + path.Join(account, "venues", venue.String(), "tickertape"))
	if err != nil {
		return nil, err
	}

	return &TickListener{
		url:      u,
		close:    make(chan struct{}),
		messages: make(chan *TickMessage, 100),
	}, nil
}

func (c *sfclient) StockTicker(account string, venue Venue, stock Symbol) (*TickListener, error) {
	u, err := url.Parse(c.baseWSURL + path.Join(account, "venues", venue.String(), "tickertape", "stocks", stock.String()))
	if err != nil {
		return nil, err
	}

	return &TickListener{
		url:      u,
		close:    make(chan struct{}),
		messages: make(chan *TickMessage, 100),
	}, nil
}

type FillMessage struct {
	APIResponse

	// Trading account of the participant this execution is for
	Account    string        `json:"account"`
	Venue      Venue         `json:"venue"`
	Symbol     Symbol        `json:"symbol"`
	Order      OrderResponse `json:"order"`
	StandingID int64         `json:"standingId"`
	IncomingID int64         `json:"incomingId"`
	Price      int64         `json:"price"`
	Filled     int64         `json:"filled"`
	FilledAt   time.Time     `json:"filledAt"`

	// Whether the order that was on the book is now complete
	StandingComplete bool `json:"standingComplete"`

	// Whether the incoming order is complete (as of this execution)
	IncomingComplete bool `json:"incomingComplete"`
}

type FillListener struct {
	url      *url.URL
	close    chan struct{}
	messages chan *FillMessage
}

func (t *FillListener) Listen() (<-chan *FillMessage, error) {
	c, _, err := websocket.DefaultDialer.Dial(t.url.String(), nil)
	if err != nil {
		return nil, err
	}

	go func() {
		defer close(t.messages)
		for {
			select {
			case <-t.close:
				// user specified close
				c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				c.Close()
				return
			default:
				msg := &FillMessage{}
				err := c.ReadJSON(msg)
				if err != nil {
					// Probably a connection error. Attempt reconnect
					log.Printf("websocket read err: %v", err)
					c.Close()
					c, _, err = websocket.DefaultDialer.Dial(t.url.String(), nil)
					if err != nil {
						log.Printf("websocket reconnect fail: %v", err)
						return
					}
					// Reconnect success
				} else {
					// Send message
					t.messages <- msg
				}
			}
		}
	}()

	return t.messages, nil
}

func (t *FillListener) Close() {
	t.close <- struct{}{}
}

func (c *sfclient) VenueFills(account string, venue Venue) (*FillListener, error) {
	u, err := url.Parse(c.baseWSURL + path.Join(account, "venues", venue.String(), "executions"))
	if err != nil {
		return nil, err
	}

	return &FillListener{
		url:      u,
		close:    make(chan struct{}),
		messages: make(chan *FillMessage, 100),
	}, nil
}

func (c *sfclient) StockFills(account string, venue Venue, stock Symbol) (*FillListener, error) {
	u, err := url.Parse(c.baseWSURL + path.Join(account, "venues", venue.String(), "executions", "stocks", stock.String()))
	if err != nil {
		return nil, err
	}

	return &FillListener{
		url:      u,
		close:    make(chan struct{}),
		messages: make(chan *FillMessage, 100),
	}, nil
}

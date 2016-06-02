package sfclient

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"time"
	"errors"
)

type Venue string

func (v Venue) String() string {
	return string(v)
}

type Symbol string

func (s Symbol) String() string {
	return string(s)
}

type OrderType string

const (
	TypeLimit             OrderType = "limit"
	TypeMarket                      = "market"
	TypeFillOrKill                  = "fill-or-kill"
	TypeImmediateOrCancel           = "immediate-or-cancel"
)

type starTransport struct {
	apiKey string
	http.RoundTripper
}

func (s *starTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("X-Starfighter-Authorization", s.apiKey)
	return s.RoundTripper.RoundTrip(r)
}

func New(apiKey string) *Client {
	return &Client{
		baseURL:   "https://api.stockfighter.io/ob/api/",
		baseWSURL: "wss://api.stockfighter.io/ob/api/ws/",
		client: &http.Client{
			Transport: &starTransport{apiKey: apiKey, RoundTripper: http.DefaultTransport},
		}}
}

type Client struct {
	baseURL   string
	baseWSURL string
	client    *http.Client
}

func unmarshalResp(body io.Reader, reply maybeErr) error {
	bodyBytes, err := ioutil.ReadAll(body)

	err = json.Unmarshal(bodyBytes, reply)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) get(endpoint string, reply maybeErr) error {
	resp, err := c.client.Get(c.baseURL + endpoint)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	err = unmarshalResp(resp.Body, reply)
	return coalesceErr( err, reply)
}

func (c *Client) postJSON(endpoint string, payload interface{}, reply maybeErr) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := c.client.Post(c.baseURL+endpoint, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	return unmarshalResp(resp.Body, reply)
}

func (c *Client) del(endpoint string, reply maybeErr) error {
	req, err := http.NewRequest("DELETE", c.baseURL+endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return unmarshalResp(resp.Body, reply)
}

type maybeErr interface {
	Err() error
}

func coalesceErr(err error, reply maybeErr) error {
	if err != nil {
		return err
	}

	return reply.Err()
}

type APIResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

func (a *APIResponse) Err() error {
	if a == nil {
		return nil
	}

	if !a.OK {
		return errors.New(a.Error)
	}

	return nil
}

type HeartbeatResponse struct {
	APIResponse
}

func (c *Client) Heartbeat() (*HeartbeatResponse, error) {
	hr := &HeartbeatResponse{}
	err := c.get("heartbeat", hr)
	if err != nil {
		return nil, err
	}

	return hr, nil
}

type VenueHeartbeatResponse struct {
	APIResponse
	Venue Venue `json:"venue"`
}

func (c *Client) VenueHeartbeat(v Venue) (*VenueHeartbeatResponse, error) {
	vhr := &VenueHeartbeatResponse{}
	err := c.get(path.Join("venues", v.String(), "heartbeat"), vhr)
	if err != nil {
		return nil, err
	}

	return vhr, nil
}

type VenueStocksResponse struct {
	APIResponse
	Symbols []struct {
		Name   string `json:"name"`
		Symbol Symbol `json:"symbol"`
	} `json:"symbols"`
}

func (c *Client) VenueStocks(v Venue) (*VenueStocksResponse, error) {
	vsr := &VenueStocksResponse{}
	err := c.get(path.Join("venues", v.String(), "stocks"), vsr)

	if err != nil {
		return nil, err
	}

	return vsr, nil
}

type AskBid struct {
	Price    int  `json:"price"`
	Quantity int  `json:"qty"`
	IsBuy    bool `json:"isBuy"`
}

type StockOrderBookResponse struct {
	APIResponse
	Venue     Venue     `json:"venue"`
	Symbol    Symbol    `json:"symbol"`
	Bids      []AskBid  `json:"bids"`
	Asks      []AskBid  `json:"asks"`
	Timestamp time.Time `json:"ts"`
}

func (c *Client) StockOrderBook(v Venue, s Symbol) (*StockOrderBookResponse, error) {
	sor := &StockOrderBookResponse{}
	err := c.get(path.Join("venues", v.String(), "stocks", s.String()), sor)
	if err != nil {
		return nil, err
	}

	return sor, nil
}

type orderRequest struct {
	Account   string    `json:"account"`
	Venue     Venue     `json:"venue"`
	Stock     Symbol    `json:"stock"`
	Price     int       `json:"price"`
	Quantity  int       `json:"qty"`
	Direction string    `json:"direction"`
	OrderType OrderType `json:"orderType"`
}

type OrderResponse struct {
	APIResponse
	Symbol           Symbol `json:"symbol"`
	Venue            Venue  `json:"venue"`
	Direction        string `json:"direction"`
	OriginalQuantity int    `json:"originalQty"`

	// This is the quantity *left outstanding*
	Quantity int `json:"qty"`

	// The price on the order -- may not match that of fills!
	Price     int    `json:"price"`
	OrderType string `json:"orderType"`

	// Guaranteed unique *on this venue*
	ID      int    `json:"id"`
	Account string `json:"account"`

	// ISO-8601 timestamp for when the order was received
	Timestamp time.Time `json:"ts"`

	// Zero, or multiple fills
	Fills       []AskBid `json:"fills"`
	TotalFilled int      `json:"totalFilled"`
	Open        bool     `json:"open"`
}

func (c *Client) postOrder(req *orderRequest) (*OrderResponse, error) {
	or := &OrderResponse{}
	err := c.postJSON(path.Join("venues", req.Venue.String(), "stocks", req.Stock.String(), "orders"), req, or)
	if err != nil {
		return nil, err
	}

	return or, nil
}

func (c *Client) BuyOrder(
	account string,
	venue Venue,
	stock Symbol,
	price int,
	quantity int,
	orderType OrderType) (*OrderResponse, error) {
	req := &orderRequest{
		Account:   account,
		Venue:     venue,
		Stock:     stock,
		Price:     price,
		Quantity:  quantity,
		Direction: "buy",
		OrderType: orderType,
	}

	return c.postOrder(req)
}

func (c *Client) SellOrder(
	account string,
	venue Venue,
	stock Symbol,
	price int,
	quantity int,
	orderType OrderType) (*OrderResponse, error) {
	req := &orderRequest{
		Account:   account,
		Venue:     venue,
		Stock:     stock,
		Price:     price,
		Quantity:  quantity,
		Direction: "sell",
		OrderType: orderType,
	}

	return c.postOrder(req)
}

type StockState struct {
	Symbol Symbol `json:"symbol"`
	Venue  Venue  `json:"venue"`

	// Best price currently bid for the stock
	Bid int `json:"bid"`

	// Best price currently offered for the stock
	Ask int `json:"ask"`

	// Aggregate size of all orders at the best bid
	BidSize int `json:"bidSize"`

	// Aggregate size of all orders at the best ask
	AskSize int `json:"askSize"`

	// Aggregate size of *all bids*
	BidDepth int `json:"bidDepth"`

	// Aggregate size of *all asks*
	AskDepth int `json:"askDepth"`

	// Price of the last trade
	Last int `json:"last"`

	// Quantity of the last trade
	LastSize int `json:"lastSize"`

	// Timestamp of the last trade
	LastTrade time.Time `json:"lastTrade"`

	// The server side timestamp the quote was last updated
	QuoteTime time.Time `json:"quoteTime"`
}

type QuoteResponse struct {
	APIResponse
	StockState
}

func (c *Client) Quote(venue Venue, stock Symbol) (*QuoteResponse, error) {
	qr := &QuoteResponse{}
	err := c.get(path.Join("venues", venue.String(), "stocks", stock.String(), "quote"), qr)
	if err != nil {
		return nil, err
	}

	return qr, nil
}

type OrderState struct {
	Symbol           Symbol `json:"symbol"`
	Venue            Venue  `json:"venue"`
	Direction        string `json:"direction"`
	OriginalQuantity int    `json:"originialQty"`

	// If this is a response to a cancel order, this will always be 0
	Quantity    int       `json:"qty"`
	Price       int       `json:"price"`
	OrderType   OrderType `json:"orderType"`
	ID          int       `json:"id"`
	Account     string    `json:"account"`
	Timestamp   time.Time `json:"ts"`
	Fills       []AskBid  `json:"fills"`
	TotalFilled int       `json:"totalFilled"`
	Open        bool      `json:"open"`
}

type StatusResponse struct {
	APIResponse
	OrderState
}

func (c *Client) OrderStatus(venue Venue, stock Symbol, id int) (*StatusResponse, error) {
	sr := &StatusResponse{}
	err := c.get(path.Join("venues", venue.String(), "stocks", stock.String(), "orders", strconv.Itoa(id)), sr)
	if err != nil {
		return nil, err
	}

	return sr, nil
}

type CancelOrderResponse StatusResponse

func (c *Client) CancelOrder(venue Venue, stock Symbol, id int) (*CancelOrderResponse, error) {
	cor := &CancelOrderResponse{}
	err := c.del(path.Join("venues", venue.String(), "stocks", stock.String(), "orders", strconv.Itoa(id)), cor)
	if err != nil {
		return nil, err
	}

	return cor, nil
}

type MultiStatusResponse struct {
	APIResponse
	Orders []OrderState `json:"orders"`
}

func (c *Client) VenueOrdersStatus(account string, venue Venue) (*MultiStatusResponse, error) {
	vr := &MultiStatusResponse{}
	err := c.get(path.Join("venues", venue.String(), "accounts", account, "orders"), vr)
	if err != nil {
		return nil, err
	}

	return vr, nil
}

func (c *Client) StockOrdersStatus(account string, venue Venue, stock Symbol) (*MultiStatusResponse, error) {
	mr := &MultiStatusResponse{}
	err := c.get(path.Join("venues", venue.String(), "accounts", account, "stocks", stock.String(), "orders"), mr)
	if err != nil {
		return nil, err
	}
	return mr, nil
}

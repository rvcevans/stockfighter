package sfclient

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"strconv"
	"time"
)

type Venue string

func (v Venue) String() string {
	return string(v)
}

const (
	VenueTESTEX Venue = "TESTEX"
)

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
	log.Printf("%s %s", r.Method, r.URL)
	return s.RoundTripper.RoundTrip(r)
}

func New(apiKey string) *sfclient {
	return &sfclient{
		baseURL: "https://api.stockfighter.io/ob/api/",
		client: &http.Client{
			Transport: &starTransport{apiKey: apiKey, RoundTripper: http.DefaultTransport},
		}}
}

type sfclient struct {
	baseURL string
	client  *http.Client
}

func unmarshalResp(body io.Reader, reply interface{}) error {
	bodyBytes, err := ioutil.ReadAll(body)

	err = json.Unmarshal(bodyBytes, reply)
	if err != nil {
		return err
	}

	return nil
}

func (c *sfclient) get(endpoint string, reply interface{}) error {
	resp, err := c.client.Get(c.baseURL + endpoint)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return unmarshalResp(resp.Body, reply)
}

func (c *sfclient) postJSON(endpoint string, payload interface{}, reply interface{}) error {
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

func (c *sfclient) del(endpoint string, reply interface{}) error {
	req, err := http.NewRequest("DELETE", endpoint, nil)
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

type APIResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

type HeartbeatResponse struct {
	APIResponse
}

func (c *sfclient) Heartbeat() (*HeartbeatResponse, error) {
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

func (c *sfclient) VenueHeartbeat(v Venue) (*VenueHeartbeatResponse, error) {
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

func (c *sfclient) VenueStocks(v Venue) (*VenueStocksResponse, error) {
	vsr := &VenueStocksResponse{}
	err := c.get(path.Join("venues", v.String(), "stocks"), vsr)

	if err != nil {
		return nil, err
	}

	return vsr, nil
}

type AskBid struct {
	Price    int64 `json:"price"`
	Quantity int64 `json:"qty"`
	IsBuy    bool  `json:"isBuy"`
}

type StockOrderBookResponse struct {
	APIResponse
	Venue     Venue     `json:"venue"`
	Symbol    Symbol    `json:"symbol"`
	Bids      []AskBid  `json:"bids"`
	Asks      []AskBid  `json:"asks"`
	Timestamp time.Time `json:"ts"`
}

func (c *sfclient) StockOrderBook(v Venue, s Symbol) (*StockOrderBookResponse, error) {
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
	Price     int64     `json:"price"`
	Quantity  int64     `json:"qty"`
	Direction string    `json:"direction"`
	OrderType OrderType `json:"orderType"`
}

type OrderResponse struct {
	APIResponse
	Symbol           Symbol    `json:"symbol"`
	Venue            Venue     `json:"venue"`
	Direction        string    `json:"direction"`
	OriginalQuantity int64     `json:"originalQty"`
	Quantity         int64     `json:"qty"`
	Price            int64     `json:"price"`
	OrderType        string    `json:"orderType"`
	ID               int64     `json:"id"`
	Account          string    `json:"account"`
	Timestamp        time.Time `json:"ts"`
	Fills            []AskBid  `json:"fills"`
	TotalFilled      int64     `json:"totalFilled"`
	Open             bool      `json:"open"`
}

func (c *sfclient) postOrder(req *orderRequest) (*OrderResponse, error) {
	or := &OrderResponse{}
	err := c.postJSON(path.Join("venues", req.Venue.String(), "stocks", req.Stock.String(), "orders"), req, or)
	if err != nil {
		return nil, err
	}

	return or, nil
}

func (c *sfclient) BuyOrder(account string, venue Venue, stock Symbol, price int64, quantity int64, orderType OrderType) (*OrderResponse, error) {
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

func (c *sfclient) SellOrder(account string, venue Venue, stock Symbol, price int64, quantity int64, orderType OrderType) (*OrderResponse, error) {
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

type QuoteResponse struct {
	APIResponse
	Symbol    Symbol    `json:"symbol"`
	Venue     Venue     `json:"venue"`
	Bid       int64     `json:"bid"`
	Ask       int64     `json:"ask"`
	BidSize   int64     `json:"bidSize"`
	BidDepth  int64     `json:"bidDepth"`
	AskDepth  int64     `json:"askSize"`
	Last      int64     `json:"last"`
	LastSize  int64     `json:"lastSize"`
	LastTrade time.Time `json:"lastTrade"`
	QuoteTime time.Time `json:"quoteTime"`
}

func (c *sfclient) Quote(venue Venue, stock Symbol) (*QuoteResponse, error) {
	qr := &QuoteResponse{}
	err := c.get(path.Join("venues", venue.String(), "stocks", stock.String(), "quote"), qr)
	if err != nil {
		return nil, err
	}

	return qr, nil
}

type StatusResponse struct {
	APIResponse
	Symbol           Symbol    `json:"symbol"`
	Venue            Venue     `json:"venue"`
	Direction        string    `json:"direction"`
	OriginalQuantity int64     `json:"originialQty"`
	Quantity         int64     `json:"qty"`
	Price            int64     `json:"price"`
	OrderType        OrderType `json:"orderType"`
	ID               int64     `json:"id"`
	Account          string    `json:"account"`
	Timestamp        time.Time `json:"ts"`
	Fills            []AskBid  `json:"fills"`
	TotalFilled      int64     `json:"totalFilled"`
	Open             bool      `json:"open"`
}

func (c *sfclient) OrderStatus(venue Venue, stock Symbol, id int64) (*StatusResponse, error) {
	sr := &StatusResponse{}
	err := c.get(path.Join("venues", venue.String(), "stocks", stock.String(), "orders", strconv.FormatInt(id, 10)), sr)
	if err != nil {
		return nil, err
	}

	return sr, nil
}

type CancelOrderResponse StatusResponse

func (c *sfclient) CancelOrder(venue Venue, stock Symbol, id int64) (*CancelOrderResponse, error) {
	cor := &CancelOrderResponse{}
	err := c.del(path.Join("venues", venue.String(), "stocks", stock.String(), "orders", strconv.FormatInt(id, 10)), cor)
	if err != nil {
		return nil, err
	}

	return cor, nil
}

type MultiStatusResponse struct {
	APIResponse
	Orders []StatusResponse `json:"orders"`
}

func (c *sfclient) VenueOrdersStatus(account string, venue Venue) (*MultiStatusResponse, error) {
	vr := &MultiStatusResponse{}
	err := c.get(path.Join("venues", venue.String(), "accounts", account, "orders"), vr)
	if err != nil {
		return nil, err
	}

	return vr, nil
}

func (c *sfclient) StockOrdersStatus(account string, venue Venue, stock Symbol) (*MultiStatusResponse, error) {
	mr := &MultiStatusResponse{}
	err := c.get(path.Join("venues", venue.String(), "accounts", account, "stocks", stock.String(), "orders"), mr)
	if err != nil {
		return nil, err
	}
	return mr, nil
}


package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/ifross89/stockfighter/sfclient"
)

var (
	apiKey  string
	account string
	stk   string
	vnu   string
)

func init() {
	flag.StringVar(&apiKey, "apikey", "", "API key to use for authentication")
	flag.StringVar(&account, "account", "", "account to do trades with")
	flag.StringVar(&stk, "stock", "", "stock to trade with")
	flag.StringVar(&vnu, "venue", "", "venue to trade at")
}

func checkempty(pairs ...string) error {
	if len(pairs)%2 != 0 {
		panic("checkempty must be called with pairs")
	}

	for i := 0; i < len(pairs); i += 2 {
		if pairs[i+1] == "" {
			return fmt.Errorf("parameter %q must be provided", pairs[i])
		}
	}
	return nil
}

func main() {
	flag.Parse()
	stock := sfclient.Symbol(stk)
	venue := sfclient.Venue(vnu)

	if err := checkempty("apikey", apiKey, "account", account, "stock", stk, "venue", vnu); err != nil {
		log.Fatalf("could not start: %v", err)
	}

	c := sfclient.New(apiKey)

	resp, err := c.StockOrderBook(venue, stock)
	if err != nil {
		log.Fatal("could not get order book for stock")
	}

	var priceToBuy int

	if len(resp.Asks) > 0 {
		priceToBuy = resp.Asks[0].Price
	} else {
		log.Fatal("no asks on the order book")
	}

	_, err = c.BuyOrder(account, venue, stock, priceToBuy, 100, sfclient.TypeLimit)
	if err != nil {
		log.Fatalf("could not execute buy: %v", err)
	}
}

package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/ifross89/stockfighter/sfclient"
)

var apiKey string

func init() {
	flag.StringVar(&apiKey, "apikey", "", "api key to use for authentication")
}

func main() {
	flag.Parse()
	if apiKey == "" {
		log.Fatal("Please provide an API key")
	}
	c := sfclient.New(apiKey)

	hr, err := c.Heartbeat()

	if err != nil {
		fmt.Printf("error during heartbeat: %v\n", err)
		return
	}

	fmt.Printf("%+v\n", hr)

	venue := sfclient.VenueTESTEX

	vhr, err := c.VenueHeartbeat(venue)

	fmt.Printf("%+v, err=%v\n", vhr, err)

	vsr, err := c.VenueStocks(venue)
	fmt.Printf("%+v, err=%v\n", vsr, err)

	for _, s := range vsr.Symbols {
		res, err := c.StockOrderBook(venue, s.Symbol)
		if err != nil {
			fmt.Printf("error getting order book for %s: err=%v", s.Symbol, err)
			continue
		}
		fmt.Printf("%+v\n", res)
	}

	resp, err := c.BuyOrder("EXB123456", venue, sfclient.Symbol("FOOBAR"), 10, 1000, sfclient.TypeFillOrKill)
	fmt.Println(resp, err)
}

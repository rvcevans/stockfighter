package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/ifross89/stockfighter/sfclient"
	"time"
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
// WIP

func main() {
	flag.Parse()
	stock := sfclient.Symbol(stk)
	venue := sfclient.Venue(vnu)

	if err := checkempty("apikey", apiKey, "account", account, "stock", stk, "venue", vnu); err != nil {
		log.Fatalf("could not start: %v", err)
	}

	c := sfclient.New(apiKey)

	hub, err := sfclient.NewStockHub(c, account, venue, stock)
	if err != nil {
		log.Fatalf("error creating hub: %v", err)
	}

	bidasks := sfclient.NewBidAskHistory(20000)

	hub.RegisterComponenets(bidasks)

	stocksToBuy := 100000

	for stocksToBuy >= 0 {
		avgAsk,_ := bidasks.Avg(20000)
		_, _, bid, bidSize := bidasks.Current()
		log.Printf("average ask:%d\tcurrent bid=%dp size%d", avgAsk, bid, bidSize)

		// If current buying price is below average asking price, SELL SELL SELL
		if bid > avgAsk && bidSize > 0 {
			log.Printf("executing bid for %d at %d", bidSize, bid)
			resp, err := hub.BuyIOC(bid, bidSize)
			if err != nil {
				log.Printf("error selling: %v", err)
			}
			log.Printf("%d filled", resp.Quantity)
			stocksToBuy -= resp.Quantity
		}
		time.Sleep(100*time.Millisecond)
	}
}

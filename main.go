package main

import (
	"flag"
	"log"

	"github.com/ifross89/stockfighter/sfclient"
)

var apiKey string
var account = "HAT11711279"
var stock = sfclient.Symbol("EPC")
var venue = sfclient.Venue("FCNEX")

func init() {
	flag.StringVar(&apiKey, "apikey", "", "api key to use for authentication")
}

func main() {
	flag.Parse()

	if apiKey == "" {
		log.Fatal("please provide an API KEY with --apikey")
	}

	mm, err := New(apiKey, account, venue, stock, 1000)
	if err != nil {
		panic(err)
	}

	mm.exec()
}

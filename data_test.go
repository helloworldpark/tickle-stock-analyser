package main

import (
	"fmt"
	"testing"
)

func TestGetPrice(t *testing.T) {
	InitDB("/Users/shp/Documents/projects/tickle-stock-watcher/credee.json")
	defer CloseDB()
	prices, needCrawling := GetPrice("028260")
	fmt.Println("Prices: ", prices)
	if needCrawling == true {
		t.FailNow()
	}
}

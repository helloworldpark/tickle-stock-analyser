package main

import (
	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
)

var dbClient *database.DBClient

func InitDB(credPath string) {
	credential := database.LoadCredential(credPath)
	dbClient = database.CreateClient()
	dbClient.Init(credential)
	dbClient.Open()
	dbClient.RegisterStructFromRegisterables([]database.DBRegisterable{
		structs.Stock{},
	})
}

func CloseDB() {
	dbClient.Close()
}

func GetStocks(markets []string) []structs.Stock {
	if len(markets) == 0 {
		markets = []string{structs.KOSPI, structs.KOSDAQ}
	}
	stocks := make([][]structs.Stock, len(markets))
	stockLen := 0
	for i := range markets {
		dbClient.Select(&(stocks[i]), "where MarketType=?", markets[i])
		stockLen += len(stocks[i])
	}
	result := make([]structs.Stock, stockLen)
	idx := 0
	for i := range stocks {
		for j := range stocks[i] {
			result[idx] = stocks[i][j]
			idx++
		}
	}
	return result
}

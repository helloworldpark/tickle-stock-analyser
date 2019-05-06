package main

import (
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/database"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
)

var dbClient *database.DBClient

func InitDB(credPath string) {
	credential := database.LoadCredential(credPath)
	dbClient = database.CreateClient()
	dbClient.Init(credential)
	dbClient.Open()
	dbClient.RegisterStructFromRegisterables([]database.DBRegisterable{
		structs.Stock{},
		structs.WatchingStock{},
		structs.StockPrice{},
	})
}

func CloseDB() {
	dbClient.Close()
}

func CreateWatcher() *watcher.Watcher {
	crawler := watcher.New(dbClient, 500*time.Millisecond)
	return crawler
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

func GetPrice(stockid string) (result []structs.StockPrice, needCrawler bool) {
	dbClient.Select(&result, "where StockID=? order by Timestamp", stockid)
	year := commons.Now().Year()
	if len(result) == 0 || result[0].Timestamp > watcher.GetCollectionStartingDate(year-2).Unix() {
		result = nil
		needCrawler = true
	}
	return result, needCrawler
}

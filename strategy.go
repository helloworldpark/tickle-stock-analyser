package main

import (
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/analyser"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
)

func GetAnalyser(stockid string, strategies [][2]string, callback func(currentTime time.Time, price float64, stockid string, orderSide int)) *analyser.Analyser {
	a := analyser.NewAnalyser(stockid)
	for i := range strategies {
		for j := range strategies[i] {
			userStock := structs.UserStock{
				StockID:   stockid,
				Strategy:  strategies[i][j],
				OrderSide: j,
			}
			f := func(currentTime time.Time, price float64, stockid string, orderSide int, userid int64, repeat bool) {
				callback(currentTime, price, stockid, orderSide)
			}
			a.AppendStrategy(userStock, f)
		}
	}
	return a
}

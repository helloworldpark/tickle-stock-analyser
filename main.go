package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/watcher"

	"github.com/helloworldpark/tickle-stock-watcher/structs"
)

func main() {
	objective := ReadAnalysisObjective("/Users/shp/Documents/projects/tickle-stock-analyser/analysisObjective.json")

	InitDB("/Users/shp/Documents/projects/tickle-stock-watcher/credee.json")
	defer CloseDB()
	// 주식 종목들을 모아놓는다
	stocks := GetStocks(objective.Markets)
	// 전략들을 파싱하여 Rule로 만들어놓는다

	// Goroutine 시작
	// 각 종목별로, 가격을 모아놓는다
	// 만일 가격이 충분하면, 일단 시뮬레이션을 시작한다
	// 가격이 충분하지 않으면, 크롤링으로 가격을 모아놓고 다시 가격을 가져와서 시뮬레이션을 시작한다

	// 각 종목별로, 일별로 가격을 밀어넣으면서 전략의 성공 여부를 매일 평가한다
	// 만일, 전략의 성공조건에 부합하게 된다면
	// 이벤트를 기록한다
	subReports := make(chan AnalysisSubReport)
	semaphore := sync.WaitGroup{}
	maxCrawlingStocks := 16
	var crawler *watcher.Watcher
	var crawled []structs.Stock
	simulate := func(stock structs.Stock, prices []structs.StockPrice) {
		defer semaphore.Done()
		subReports <- Simulate(stock, objective.Rules, prices)
	}

	go func() {
		// 모든 데이터를 소진했으면, 이벤트를 기록한다
		for sub := range subReports {
			fmt.Println("222222: ", sub)
		}
	}()

	for stockIdx := range stocks {
		stock := stocks[stockIdx]
		if stockIdx%maxCrawlingStocks == 0 {
			crawler = CreateWatcher()
			crawled = make([]structs.Stock, 0)
		}
		prices, needCrawling := GetPrice(stock.StockID)
		if needCrawling {
			crawler.Register(stock)
			crawled = append(crawled, stock)
		} else {
			semaphore.Add(1)
			go simulate(stock, prices)
		}

		if stockIdx%maxCrawlingStocks == (maxCrawlingStocks-1) || stockIdx == len(stocks)-1 {
			crawler.Collect()
			semaphore.Add(len(crawled))
			for _, crawledStock := range crawled {
				prices, needCrawling = GetPrice(stock.StockID)
				if needCrawling {
					semaphore.Done()
				} else {
					go simulate(crawledStock, prices)
				}
			}
		}
	}

	semaphore.Wait()
	fmt.Println("Will Close SubReports")
	close(subReports)

	// 기록된 이벤트를 Google Cloud Storage에 저장한다
	// Goroutine 종료

	// 모든 종목에 대해 종료되었으면, 이 서버를 삭제하는 API를 날리고 종료한다
}

func Simulate(stock structs.Stock, strategies [][2]string, prices []structs.StockPrice) AnalysisSubReport {
	lastSide := -1
	callback := func(currentTime time.Time, price float64, stockid string, orderSide int) {
		if lastSide == -1 {
			if orderSide == 1 {
				return
			}
			lastSide = orderSide
		} else {
			if lastSide == orderSide {
				return
			}
		}

	}
	ana := GetAnalyser(stock.StockID, strategies, callback)
	for i := range prices {
		price := prices[i]
		ana.AppendPastStockPrice(price)
		ana.CalculateStrategies()
	}

	return NewSubReport(stock, nil)
}

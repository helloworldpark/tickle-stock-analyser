package main

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/helloworldpark/tickle-stock-watcher/analyser"
	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/helloworldpark/tickle-stock-watcher/watcher"
	"github.com/sdcoffey/techan"

	"github.com/helloworldpark/tickle-stock-watcher/structs"
)

const (
	maxCrawlingStocks = 8
	INVALID           = -1
	BUY               = 0
	SELL              = 1
)

func main() {
	timestampStart := commons.Now().Unix()
	// 분석할 방법과 대상을 로드
	objective := ReadAnalysisObjective("/Users/shp/Documents/projects/tickle-stock-analyser/analysisObjective2.json")

	// DB 초기화
	InitDB("/Users/shp/Documents/projects/tickle-stock-watcher/credee.json")
	defer CloseDB()

	// Storage 초기화
	InitStorage()

	// 주식 종목들을 모아놓는다
	stocks := GetStocks(objective.Markets)

	// Goroutine 시작
	// 각 종목별로, 가격을 모아놓는다
	// 만일 가격이 충분하면, 일단 시뮬레이션을 시작한다
	// 가격이 충분하지 않으면, 크롤링으로 가격을 모아놓고 다시 가격을 가져와서 시뮬레이션을 시작한다
	// 전략들을 파싱하여 Rule로 만들어놓는다
	// 각 종목별로, 일별로 가격을 밀어넣으면서 전략의 성공 여부를 매일 평가한다
	// 만일, 전략의 성공조건에 부합하게 된다면 이벤트를 메모리상에 기록해둔다
	subReports := make(chan []AnalysisSubReport)
	simulateSemaphore := sync.WaitGroup{}
	reportSemaphore := sync.WaitGroup{}
	var crawler *watcher.Watcher
	var crawled []structs.Stock
	simulate := func(stock structs.Stock, prices []structs.StockPrice) {
		defer simulateSemaphore.Done()
		subReports <- Simulate(stock, objective.Rules, prices)
	}

	// 모든 데이터를 소진했으면, 이벤트를 기록한다
	go func() {
		defer reportSemaphore.Done()

		subs := make([]AnalysisSubReport, 0)
		for sub := range subReports {
			for k := range sub {
				subs = append(subs, sub[k])
			}
		}
		timestampEnd := commons.Now().Unix()

		report := AnalysisReport{
			TimestampStart: timestampStart,
			TimestampEnd:   timestampEnd,
			Reports:        subs,
		}
		fileName, err := Write(report, runtime.GOOS == "darwin")
		if err != nil {
			logger.Error(err.Error())
		} else {
			logger.Info("[Writer] Writed report: %d subreports at %s", len(subs), fileName)
		}
	}()
	reportSemaphore.Add(1)

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
			simulateSemaphore.Add(1)
			go simulate(stock, prices)
		}

		if len(crawled) > 0 && (stockIdx%maxCrawlingStocks == (maxCrawlingStocks-1) || stockIdx == len(stocks)-1) {
			crawler.Collect()
			simulateSemaphore.Add(len(crawled))
			for _, crawledStock := range crawled {
				prices, needCrawling = GetPrice(stock.StockID)
				if needCrawling {
					simulateSemaphore.Done()
				} else {
					go simulate(crawledStock, prices)
				}
			}
		}
	}
	// 시뮬레이션이 끝나길 기다렸다가, 다 끝나면 보고서를 작성한다
	simulateSemaphore.Wait()
	close(subReports)

	// 기록된 이벤트를 Google Cloud Storage에 저장한다
	// Goroutine 종료
	reportSemaphore.Wait()

	logger.Info("Finished Analysing on %s!", runtime.GOOS)
	// if runtime.GOOS == "linux" {
	// 	// 서버를 날리자꾸나
	// } else if runtime.GOOS == "darwin" {

	// }
}

func TestStocks() []structs.Stock {
	return []structs.Stock{
		structs.Stock{Name: "대현", StockID: "016090", MarketType: "kospi"},
		structs.Stock{Name: "대한항공", StockID: "003490", MarketType: "kospi"},
		structs.Stock{Name: "한미사이언스", StockID: "008930", MarketType: "kospi"},
		structs.Stock{Name: "삼성물산", StockID: "028260", MarketType: "kospi"},
		structs.Stock{Name: "한화케미칼", StockID: "009830", MarketType: "kospi"},
		structs.Stock{Name: "동서", StockID: "026960", MarketType: "kospi"},
		structs.Stock{Name: "CJ CGV", StockID: "079160", MarketType: "kospi"},
	}
}

func Simulate(stock structs.Stock, strategies [][2]string, prices []structs.StockPrice) []AnalysisSubReport {
	subReports := make([]AnalysisSubReport, len(strategies))
	startTimestamp, endTimestamp := prices[0].Timestamp, prices[len(prices)-1].Timestamp
	for i := range subReports {
		ana := GetAnalyser(stock.StockID)

		strategyCallbacks := [2]StrategyCallback{}
		trades := make([]Trade, 0)

		sc0 := StrategyCallback{}
		sc0.strategy = strategies[i][0]
		if strategies[i][1] == "" {
			sc0.callback = generate4percentCallback(&trades, stock, ana)
		} else {
			sc0.callback = generateBaseCallback(&trades)
		}

		sc1 := StrategyCallback{}
		sc1.strategy = strategies[i][1]
		if sc1.strategy == "" {
			sc1.strategy = "price()>=0"
		}
		sc1.callback = sc0.callback

		strategyCallbacks[0] = sc0
		strategyCallbacks[1] = sc1
		UpdateStrategy(ana, stock.StockID, strategyCallbacks)

		for j := range prices {
			ana.AppendPastPrice(prices[j])
			ana.CalculateStrategies()
		}

		if len(trades) > 0 {
			if (trades[len(trades)-1].Sell == structs.StockPrice{}) {
				trades = trades[:len(trades)-1]
			}
		}

		sellStrategy := strategies[i][1]
		if sellStrategy == "" {
			sellStrategy = "price()>=buy*1.02"
		}
		strategy := fmt.Sprintf("BUY:%s SELL:%s", strategies[i][0], sellStrategy)
		logger.Info("Strategy: %s", strategy)
		subReports[i] = NewSubReport(stock, strategy, trades, startTimestamp, endTimestamp)
	}
	tradeCount0 := len(subReports[0].Trades)
	tradeCount1 := len(subReports[1].Trades)
	logger.Info("[Simulate] Finished %s: %d scenarios(0: %d trades, 1: %d trades)", stock.StockID, len(subReports), tradeCount0, tradeCount1)
	return subReports
}

func generateBaseCallback(trades *[]Trade) func(structs.StockPrice, int) {
	lastSide := INVALID
	callback := func(price structs.StockPrice, orderSide int) {
		if lastSide == INVALID {
			if orderSide == SELL {
				return
			}
			lastSide = orderSide
		} else {
			if lastSide == orderSide {
				return
			}
			lastSide = orderSide
		}
		if lastSide == BUY {
			trade := Trade{}
			trade.Buy = price
			*trades = append(*trades, trade)
		} else if lastSide == SELL {
			trade := (*trades)[len(*trades)-1]
			trade.Sell = price
			(*trades)[len(*trades)-1] = trade
		}
	}
	return callback
}

func generate4percentCallback(trades *[]Trade, stock structs.Stock, a *analyser.Analyser) func(structs.StockPrice, int) {
	lastSide := INVALID
	idx := 0
	var callback func(price structs.StockPrice, orderSide int)
	callback = func(price structs.StockPrice, orderSide int) {
		if lastSide == INVALID {
			if orderSide == SELL {
				return
			}
			lastSide = orderSide
		} else {
			if lastSide == orderSide {
				return
			}
			lastSide = orderSide
		}
		idx++
		if lastSide == BUY {
			trade := Trade{}
			trade.Buy = price
			*trades = append(*trades, trade)

			a.DeleteStrategy(0, techan.SELL)
			persent4 := fmt.Sprintf("price()>=%f", float64(price.Close)*1.04)
			userStock := structs.UserStock{
				StockID:   stock.StockID,
				Strategy:  persent4,
				OrderSide: 1,
			}
			a.AppendStrategy(userStock, func(price structs.StockPrice, orderSide int, _ int64, _ bool) {
				callback(price, orderSide)
			})
		} else if lastSide == SELL {
			trade := (*trades)[len(*trades)-1]
			trade.Sell = price
			(*trades)[len(*trades)-1] = trade

			a.DeleteStrategy(0, techan.SELL)
		}
	}
	return callback
}

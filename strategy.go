package main

import (
	"encoding/json"
	"io/ioutil"
	"math"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/analyser"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
	"gonum.org/v1/gonum/stat"
)

type AnalysisObjective struct {
	Rules   [][2]string `json:"rules"`
	Markets []string    `json:"markets"`
}

type Trade struct {
	Buy  structs.StockPrice
	Sell structs.StockPrice
}

type AnalysisSubReport struct {
	Stock         structs.Stock `json:"stock"`
	Trades        []Trade       `json:"trades"`
	Count         int           `json:"count"`
	ProfitMean    float64       `json:"profit_mean"`
	ProfitVar     float64       `json:"profit_var"`
	ProfitLogMean float64       `json:"profit_log_mean"`
	ProfitLogVar  float64       `json:"profit_log_var"`
	LagMean       float64       `json:"lag_mean"`
	LagVar        float64       `json:"lag_var"`
	LagLogMean    float64       `json:"lag_log_mean"`
	LagLogVar     float64       `json:"lag_log_var"`
}

type AnalysisReport struct {
	TimestampStart int64               `json:"timestamp_start"`
	TimestampEnd   int64               `json:"timestamp_end"`
	Reports        []AnalysisSubReport `json:"reports"`
}

func ReadAnalysisObjective(filePath string) AnalysisObjective {
	raw, err := ioutil.ReadFile(filePath)
	if err != nil {
		logger.Panic("%v", err)
	}

	var obj AnalysisObjective
	if err := json.Unmarshal(raw, &obj); err != nil {
		logger.Panic("%v", err)
	}
	return obj
}

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

func NewSubReport(stock structs.Stock, trades []Trade) AnalysisSubReport {
	subReport := AnalysisSubReport{Stock: stock, Trades: trades, Count: len(trades)}
	if len(trades) == 0 {
		return subReport
	}

	nums := make([]float64, len(trades))

	for i := range nums {
		nums[i] = float64(trades[i].Sell.Close) / float64(trades[i].Buy.Close)
	}
	pLm, pLv := logMeanVar(nums)
	pSm, pSv := simpleMeanVar(nums)
	subReport.ProfitMean = pSm - 1.0
	subReport.ProfitVar = pSv
	subReport.ProfitLogMean = pLm
	subReport.ProfitLogVar = pLv

	for i := range nums {
		nums[i] = float64(trades[i].Sell.Timestamp) / float64(trades[i].Buy.Timestamp)
	}
	lLm, lLv := logMeanVar(nums)
	lSm, lSv := simpleMeanVar(nums)
	subReport.LagMean = lSm
	subReport.LagVar = lSv
	subReport.LagLogMean = lLm
	subReport.LagLogVar = lLv

	return subReport
}

func simpleMeanVar(arr []float64) (m, v float64) {
	return stat.MeanVariance(arr, nil)
}

func logMeanVar(arr []float64) (m, v float64) {
	tmp := make([]float64, len(arr))
	for i := range tmp {
		tmp[i] = math.Log(arr[i])
	}
	return stat.MeanVariance(tmp, nil)
}

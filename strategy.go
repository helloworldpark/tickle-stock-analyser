package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"

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
	Stock          structs.Stock `json:"stock"`
	Strategy       string        `json:"strategy"`
	Trades         []Trade       `json:"trades"`
	Count          int           `json:"count"`
	StartTimestamp int64         `json:"startTimestamp"`
	EndTimestamp   int64         `json:"endTimestamp"`
	ProfitMean     float64       `json:"profit_mean"`
	ProfitVar      float64       `json:"profit_var"`
	ProfitLogMean  float64       `json:"profit_log_mean"`
	ProfitLogVar   float64       `json:"profit_log_var"`
	LagMean        float64       `json:"lag_mean"`
	LagVar         float64       `json:"lag_var"`
	LagLogMean     float64       `json:"lag_log_mean"`
	LagLogVar      float64       `json:"lag_log_var"`
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

type StrategyCallback struct {
	strategy string
	callback func(price structs.StockPrice, orderSide int)
}

func GetAnalyser(stockid string) *analyser.Analyser {
	a := analyser.NewAnalyser(stockid)
	return a
}

func UpdateStrategy(a *analyser.Analyser, stockid string, strategies [2]StrategyCallback) {
	for j := range strategies {
		userStock := structs.UserStock{
			StockID:   stockid,
			Strategy:  strategies[j].strategy,
			OrderSide: j,
		}
		f := func(c func(structs.StockPrice, int)) func(structs.StockPrice, int, int64, bool) {
			return func(price structs.StockPrice, orderSide int, _ int64, _ bool) {
				c(price, orderSide)
			}
		}
		callback := strategies[j].callback
		success, err := a.AppendStrategy(userStock, f(callback))
		if !success {
			fmt.Println("[Strategy] ", err)
		}
	}
}

func NewSubReport(stock structs.Stock, strategy string, trades []Trade, start, end int64) AnalysisSubReport {
	subReport := AnalysisSubReport{
		Stock:          stock,
		Trades:         trades,
		Strategy:       strategy,
		Count:          len(trades),
		StartTimestamp: start,
		EndTimestamp:   end,
	}
	if len(trades) == 0 {
		return subReport
	}

	nums := make([]float64, len(trades))
	for i := range nums {
		nums[i] = float64(trades[i].Sell.Close) / float64(trades[i].Buy.Close)
	}

	if len(trades) == 1 {
		subReport.ProfitMean = nums[0] - 1.0
		subReport.ProfitLogMean = math.Log(nums[0])

		subReport.LagMean = (float64(trades[0].Sell.Timestamp) - float64(trades[0].Buy.Timestamp))
		subReport.LagMean /= (60 * 60 * 24)
		subReport.LagLogMean = math.Log(subReport.LagMean)
		return subReport
	}

	pLm, pLv := logMeanVar(nums)
	pSm, pSv := simpleMeanVar(nums)
	subReport.ProfitMean = pSm - 1.0
	subReport.ProfitVar = pSv
	subReport.ProfitLogMean = pLm
	subReport.ProfitLogVar = pLv

	for i := range nums {
		nums[i] = float64(trades[i].Sell.Timestamp) - float64(trades[i].Buy.Timestamp)
		nums[i] /= (60 * 60 * 24)
	}
	lLm, lLv := logMeanVar(nums)
	lSm, lSv := simpleMeanVar(nums)
	subReport.LagMean = lSm
	subReport.LagVar = lSv
	subReport.LagLogMean = lLm
	subReport.LagLogVar = lLv

	return subReport
}

func (sr AnalysisSubReport) String() string {
	bf := bytes.Buffer{}
	bf.WriteString("Name: ")
	bf.WriteString(sr.Stock.Name)
	bf.WriteString("(")
	bf.WriteString(sr.Stock.StockID)
	bf.WriteString(")\n")
	bf.WriteString(fmt.Sprintf("Strategy: %s\n", sr.Strategy))
	bf.WriteString(fmt.Sprintf("Trades: %d\n", sr.Count))
	bf.WriteString(fmt.Sprintf("Total Profit: %.2f%%\n", sr.ProfitMean*float64(sr.Count)*100))
	bf.WriteString(fmt.Sprintf("Profit Mean: %.2f%%\n", sr.ProfitMean*100))
	bf.WriteString(fmt.Sprintf("Profit Stdev: %.2f%%\n", sr.ProfitVar*100))
	bf.WriteString(fmt.Sprintf("Profit Log Mean: %.2f%%\n", math.Expm1(sr.ProfitLogMean)*100))
	bf.WriteString(fmt.Sprintf("Profit Log Stdev: %.2f%%\n", sr.ProfitLogVar*100))
	bf.WriteString(fmt.Sprintf("Lag Mean: %.2f days\n", sr.LagMean))
	bf.WriteString(fmt.Sprintf("Lag Stdev: %.2f days\n", sr.LagVar))
	bf.WriteString(fmt.Sprintf("Lag Log Mean: %.2f days\n", math.Exp(sr.LagLogMean)))
	bf.WriteString(fmt.Sprintf("Lag Log Stdev: %.2f days\n", sr.LagLogVar))

	return bf.String()
}

func simpleMeanVar(arr []float64) (m, v float64) {
	return stat.MeanStdDev(arr, nil)
}

func logMeanVar(arr []float64) (m, v float64) {
	tmp := make([]float64, len(arr))
	for i := range tmp {
		tmp[i] = math.Log(arr[i])
	}
	return stat.MeanStdDev(tmp, nil)
}

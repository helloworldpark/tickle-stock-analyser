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

func GetAnalyser(stockid string, strategies [][2]string, callback func(price structs.StockPrice, orderSide int)) *analyser.Analyser {
	a := analyser.NewAnalyser(stockid)
	for i := range strategies {
		for j := range strategies[i] {
			userStock := structs.UserStock{
				StockID:   stockid,
				Strategy:  strategies[i][j],
				OrderSide: j,
			}
			f := func(price structs.StockPrice, orderSide int, userid int64, repeat bool) {
				callback(price, orderSide)
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
	subReport.ProfitLogMean = math.Expm1(pLm)
	subReport.ProfitLogVar = math.Exp(pLv)

	for i := range nums {
		nums[i] = float64(trades[i].Sell.Timestamp) - float64(trades[i].Buy.Timestamp)
		nums[i] /= (60 * 60 * 24)
	}
	lLm, lLv := logMeanVar(nums)
	lSm, lSv := simpleMeanVar(nums)
	subReport.LagMean = lSm
	subReport.LagVar = lSv
	subReport.LagLogMean = math.Exp(lLm)
	subReport.LagLogVar = math.Exp(lLv)

	return subReport
}

func (sr AnalysisSubReport) String() string {
	bf := bytes.Buffer{}
	bf.WriteString("Name: ")
	bf.WriteString(sr.Stock.Name)
	bf.WriteString("(")
	bf.WriteString(sr.Stock.StockID)
	bf.WriteString(")\n")
	bf.WriteString(fmt.Sprintf("Trades: %d\n", sr.Count))
	bf.WriteString(fmt.Sprintf("Total Profit: %.2f%%\n", sr.ProfitMean*float64(sr.Count)*100))
	bf.WriteString(fmt.Sprintf("Profit Mean: %.2f%%\n", sr.ProfitMean*100))
	bf.WriteString(fmt.Sprintf("Profit Stdev: %.2f%%\n", sr.ProfitVar*100))
	bf.WriteString(fmt.Sprintf("Profit Log Mean: %.2f%%\n", sr.ProfitLogMean*100))
	bf.WriteString(fmt.Sprintf("Profit Log Stdev: %.2f%%\n", sr.ProfitLogVar*100))
	bf.WriteString(fmt.Sprintf("Lag Mean: %.2f days\n", sr.LagMean))
	bf.WriteString(fmt.Sprintf("Lag Stdev: %.2f days\n", sr.LagVar))
	bf.WriteString(fmt.Sprintf("Lag Log Mean: %.2f days\n", sr.LagLogMean))
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

package main

import "fmt"
import "github.com/helloworldpark/tickle-stock-watcher/structs"

func main() {
	InitDB("/Users/shp/Documents/projects/tickle-stock-watcher/credee.json")
	defer CloseDB()
	// 주식 종목들을 모아놓는다
	stocks := GetStocks([]string{"kospi"})
	for i := range stocks {
		fmt.Println(stocks[i])
	}
	// 전략들을 파싱하여 Rule로 만들어놓는다

	// Goroutine 시작
	// 각 종목별로, 가격을 모아놓는다

	// 각 종목별로, 일별로 가격을 밀어넣으면서 전략의 성공 여부를 매일 평가한다
	// 만일, 전략의 성공조건에 부합하게 된다면
	// 이벤트를 기록한다

	// 모든 데이터를 소진했으면, 이벤트를 기록한다

	// 기록된 이벤트를 Google Cloud Storage에 저장한다
	// Goroutine 종료

	// 모든 종목에 대해 종료되었으면, 이 서버를 삭제하는 API를 날리고 종료한다
	stock := structs.Stock{Name: "name"}

	fmt.Println("Hello world! ", stock)
}

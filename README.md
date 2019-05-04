# tickle-stock-analyser
주식 가격을 분석하여 어떤 전략이 단타치기 좋은지 알아보고자 한다

## 구조

### 분석 시작
 - Google Compute Engine을 시작
 - Docker로 저장해둔 컨테이너 이미지를 시작한다
 - 이미지의 내용은 아래에 있다

### 데이터 수집
 - 데이터를 먼저 DB에서 조회해본다
     - [github.com/helloworldpark/tickle-stock-watcher/database] 를 그대로 사용
 - 데이터가 충분히 많지 않으면, 크롤러로 직접 모아놓는다
     - 데이터는 최소 2년치
     - [github.com/helloworldpark/tickle-stock-watcher/watcher] 를 그대로 사용
     - 모은 데이터는 DB에 저장

### 전략 파서
 - 인자로 받은 전략을 파싱하여 ```techan.Rule```로 변환한다
     - [github.com/helloworldpark/tickle-stock-watcher/analyser] 를 그대로 사용

### 시뮬레이션
 - 파싱하여 생성된 ```techan.Rule```을 갖고 시뮬레이션한다
 - 실제 느낌을 주기 위해 데이터는 배열로 들고 있되, 1일에 하나의 데이터만 ```analyser.Analyser```에 주도록 한다
 - 이벤트는 가격이 전략에 부합할 때 발생시킨다
 - 이벤트 시점에 아래 정보를 수집한다
     - 시가, 종가, 최고가, 최저가(price)
     - 시점(timestamp)
     - 이벤트 종류(buy/sell)
 - buy 이벤트가 발생했으면, sell 이벤트가 발생하기 전까지는 buy 이벤트를 발생시키지 않는다
 - sell 이벤트가 발생했으면, buy 이벤트가 발생하기 전까지는 sell 이벤트를 발생시키지 않는다

### 결과 분석
 - 아래 항목들을 분석한다
     - 수익률의 series
     - 수익률의 평균
     - 수익률의 분산
     - 수익 발생까지의 소요 시간의 series
     - 소요시간의 평균
     - 소요시간의 분산

### 결과 작성
 - **결과 분석**에서 나온 항목들을 JSON으로 파일에 기록한다
     - 파일명: stock_{종목이름}_{종목번호}.json
     - ```종목이름```, ```종목번호```, ```분석일자```는 맨 위에 넣는다
 - 기록된 파일은 Google Cloud Storage에 저장한다
 
 ### 종료
  - 모든 분석이 끝났으면, Google Compute Engine을 종료한다
  - Google Compute Engine을 Delete하는 REST API 콜을 날린다

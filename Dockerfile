# 1. Install golang
FROM golang:latest
LABEL email="helloworld.park@gmail.com" 

# 2. Download code
RUN cd ./src
RUN git clone https://github.com/helloworldpark/tickle-stock-analyser.git && cd ./tickle-stock-analyser && go build && ./tickle-stock-analyser

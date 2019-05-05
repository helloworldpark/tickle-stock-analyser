package main

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/storage"
	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
)

var client *storage.Client
var bucket *storage.BucketHandle

func InitStorage() {
	ctx := context.Background()
	clientYet, err := storage.NewClient(ctx)
	if err != nil {
		logger.Panic(err.Error())
	}
	client = clientYet
	bucket = client.Bucket("ticklemeta-storage")
}

func Write(report interface{}) error {
	ctx := context.Background()
	now := commons.Now().Unix()
	writer := bucket.Object(fmt.Sprintf("tickle-stock-analyser/Analysis%d.json", now)).NewWriter(ctx)
	jsonReport, err := json.Marshal(&report)
	if err != nil {
		return err
	}
	writer.Write(jsonReport)
	return writer.Close()
}

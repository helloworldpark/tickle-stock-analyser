package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

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

func Write(report interface{}, writeLocal bool) (string, error) {
	ctx := context.Background()
	now := commons.Now()
	y, m, d := now.Date()
	h, i, s := now.Clock()
	pureFileName := fmt.Sprintf("Analysis%d%d%d%d%d%d.json", y, m, d, h, i, s)
	filePath := "tickle-stock-analyser/" + pureFileName
	writer := bucket.Object(filePath).NewWriter(ctx)
	jsonReport, err := json.Marshal(&report)
	if err != nil {
		return "", err
	}
	writer.Write(jsonReport)

	if writeLocal {
		filePath = "/Users/shp/Documents/projects/tickle-stock-analyser-python/" + pureFileName
		err = ioutil.WriteFile(filePath, jsonReport, 0777)
		if err == nil {
			logger.Info("[Storage] Saved to local at " + filePath)
		} else {
			logger.Error("[Storage] Failed to save to local:" + filePath)
		}
	}
	return filePath, writer.Close()
}

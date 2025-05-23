package main

import (
	"context"

	"github.com/eser/bfo/pkg/api/adapters/appcontext"
)

func main() {
	appContext := appcontext.NewAppContext()
	baseCtx := context.Background()

	err := appContext.Init(baseCtx)
	if err != nil {
		panic(err)
	}

	result, err := appContext.Repository.ListTaskBuckets(baseCtx)
	if err != nil {
		panic(err)
	}

	for _, bucket := range result {
		appContext.Logger.InfoContext(baseCtx, "Task Bucket", "bucket", bucket)
	}

}

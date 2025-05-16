package main

import (
	"context"

	"github.com/eser/bfo/pkg/api/adapters/appcontext"
	"github.com/eser/bfo/pkg/api/adapters/http"
)

func main() {
	ctx := context.Background()

	appContext, err := appcontext.NewAppContext(ctx)
	if err != nil {
		panic(err)
	}

	err = appContext.Run(ctx)
	if err != nil {
		panic(err)
	}

	err = http.Run(
		ctx,
		appContext,
	)
	if err != nil {
		panic(err)
	}
}

package main

import (
	"context"
	_ "net/http/pprof" // #nosec G108 -- pprof is intentionally enabled for local profiling

	"efficient-request-queueing-for-llm-inference/internal/application"
)

func main() {

	rootctx, rootcancel := context.WithCancel(context.Background())
	defer rootcancel()

	app, err := application.BuildApplication(rootctx)
	if err != nil {
		panic(err)
	}
	if err := app.Run(rootctx); err != nil {
		panic(err)
	}
}

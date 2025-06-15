package main

import (
	"context"

	"go.viam.com/rdk/logging"
)

func main() {
	err := realMain()
	if err != nil {
		panic(err)
	}
}

func realMain() error {
	_ = context.Background()
	_ = logging.NewLogger("cli")

	return nil
}

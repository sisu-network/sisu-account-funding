package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sisu-network/sisu-account-funding/core"
)

func main() {
	core.Run()

	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
}

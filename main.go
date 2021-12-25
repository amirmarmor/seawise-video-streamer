package main

import (
	"os"
	"os/signal"
	"syscall"
	"www.seawise.com/client/entrypoint"
)

func main() {
	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGINT)
	go func() {
		sig := <-signalChannel
		switch sig {
		case os.Interrupt:
			os.Exit(0)
		case syscall.SIGINT:
			os.Exit(0)
		}
	}()
	e := entrypoint.EntryPoint{}
	e.Run()
}

package main

import (
	"github.com/t-kuni/grpc-example/client/domain"
	"github.com/t-kuni/grpc-example/client/presenter"
	"log"
)

func main() {
	presenter.ClearConsole()

	app := domain.NewApp()

	host := "localhost"
	port := 30000
	app.ConnectServer(host, port)

	name := presenter.ScanUserName()

	app.Join(name)

	presenter.ClearConsole()

	ui := presenter.NewUI(name)
	app.ConnectUI(ui)

	app.StartWatchState()

	err := ui.Start()
	if err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"github.com/ynoproject/ynorankings/api"
	"github.com/ynoproject/ynorankings/cli"
	"github.com/ynoproject/ynorankings/database"
	"github.com/ynoproject/ynorankings/rankings"
)

func main() {
	database.Init()
	cli.Run()
	rankings.Init()
	api.Init()
}

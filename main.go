package main

import (
	"github.com/ynoproject/ynorankings/api"
	"github.com/ynoproject/ynorankings/database"
	"github.com/ynoproject/ynorankings/rankings"
)

func main() {
	database.Init()
	rankings.Init()
	api.Init()
}

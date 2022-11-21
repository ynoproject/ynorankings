package main

import (
	"ynorankings/api"
	"ynorankings/database"
	"ynorankings/rankings"
)

func main() {
	database.Init()
	rankings.Init()
	api.Init()
}

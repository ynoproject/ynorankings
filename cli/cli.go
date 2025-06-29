package cli

import (
	"flag"
	"github.com/ynoproject/ynorankings/common"
	"github.com/ynoproject/ynorankings/database"
	"os"
)

const (
	CommandNone = iota
	CommandUpdateRankings
	CommandInvalid = -1
)

type Cli struct {
	Command     int
	CommandArgs []string
}

func Run() {
	cmd := parse()
	if cmd.Command == CommandInvalid {
		println(`Usage:
    ynorankings # launches the server
    ynorankings update-rankings <category> <subcategory> <game>`)
		flag.Usage()
		os.Exit(1)
	}

	var err error
	switch cmd.Command {
	case CommandUpdateRankings:
		common.CurrentEventPeriodOrdinal, _ = database.GetCurrentEventPeriodOrdinal()
		categoryId := cmd.CommandArgs[0]
		subcategoryId := cmd.CommandArgs[1]
		gameId := cmd.CommandArgs[2]
		err = database.UpdateRankingEntries(categoryId, subcategoryId, gameId)
	}

	if err != nil {
		println(err)
		os.Exit(1)
	} else if cmd.Command != CommandNone {
		os.Exit(0)
	}
}

func parse() (flags Cli) {
	// no flags yet
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		return
	}

	switch args[0] {
	case "update-rankings":
		if len(args[1:]) == 3 {
			flags.Command = CommandUpdateRankings
			flags.CommandArgs = args[1:]
		}
	}

	if flags.Command == CommandNone {
		flags.Command = CommandInvalid
	}

	return
}

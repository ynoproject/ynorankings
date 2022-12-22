package rankings

import (
	"log"
	"strconv"
	"time"

	"github.com/ynoproject/ynorankings/common"
	"github.com/ynoproject/ynorankings/database"

	"github.com/go-co-op/gocron"
)

var (
	scheduler = gocron.NewScheduler(time.UTC)
)

func Init() {
	common.CurrentEventPeriodOrdinal, _ = database.GetCurrentEventPeriodOrdinal()

	for _, gameName := range common.GameNames {
		var rankingCategories []*common.RankingCategory

		bpCategory := &common.RankingCategory{CategoryId: "bp"}
		rankingCategories = append(rankingCategories, bpCategory)

		badgeCountCategory := &common.RankingCategory{CategoryId: "badgeCount"}
		rankingCategories = append(rankingCategories, badgeCountCategory)

		bpCategory.SubCategories = append(bpCategory.SubCategories, common.RankingSubCategory{SubCategoryId: "all"})
		badgeCountCategory.SubCategories = append(badgeCountCategory.SubCategories, common.RankingSubCategory{SubCategoryId: "all"})

		bpCategory.SubCategories = append(bpCategory.SubCategories, common.RankingSubCategory{SubCategoryId: gameName, Game: gameName})
		badgeCountCategory.SubCategories = append(badgeCountCategory.SubCategories, common.RankingSubCategory{SubCategoryId: gameName, Game: gameName})

		eventPeriods, err := database.GetEventPeriodData(gameName)
		if err != nil {
			log.Print("SERVER", "exp", err.Error())
		} else if len(eventPeriods) > 0 {
			expCategory := &common.RankingCategory{CategoryId: "exp", Game: gameName, Periodic: true}
			rankingCategories = append(rankingCategories, expCategory)

			if len(eventPeriods) > 1 {
				expCategory.SubCategories = append(expCategory.SubCategories, common.RankingSubCategory{SubCategoryId: "all", Game: gameName})
			}
			for _, eventPeriod := range eventPeriods {
				expCategory.SubCategories = append(expCategory.SubCategories, common.RankingSubCategory{SubCategoryId: strconv.Itoa(eventPeriod.PeriodOrdinal), Game: gameName})
			}

			eventLocationCountCategory := &common.RankingCategory{CategoryId: "eventLocationCount", Game: gameName, Periodic: true}
			rankingCategories = append(rankingCategories, eventLocationCountCategory)

			if len(eventPeriods) > 1 {
				eventLocationCountCategory.SubCategories = append(eventLocationCountCategory.SubCategories, common.RankingSubCategory{SubCategoryId: "all", Game: gameName})
			}
			for _, eventPeriod := range eventPeriods {
				eventLocationCountCategory.SubCategories = append(eventLocationCountCategory.SubCategories, common.RankingSubCategory{SubCategoryId: strconv.Itoa(eventPeriod.PeriodOrdinal), Game: gameName})
			}

			freeEventLocationCountCategory := &common.RankingCategory{CategoryId: "freeEventLocationCount", Game: gameName, Periodic: true}
			rankingCategories = append(rankingCategories, freeEventLocationCountCategory)

			if len(eventPeriods) > 1 {
				freeEventLocationCountCategory.SubCategories = append(freeEventLocationCountCategory.SubCategories, common.RankingSubCategory{SubCategoryId: "all", Game: gameName})
			}
			for _, eventPeriod := range eventPeriods {
				freeEventLocationCountCategory.SubCategories = append(freeEventLocationCountCategory.SubCategories, common.RankingSubCategory{SubCategoryId: strconv.Itoa(eventPeriod.PeriodOrdinal), Game: gameName})
			}

			eventLocationCompletionCategory := &common.RankingCategory{CategoryId: "eventLocationCompletion", Game: gameName, Periodic: true}
			rankingCategories = append(rankingCategories, eventLocationCompletionCategory)

			if len(eventPeriods) > 1 {
				eventLocationCompletionCategory.SubCategories = append(eventLocationCompletionCategory.SubCategories, common.RankingSubCategory{SubCategoryId: "all", Game: gameName})
			}
			for _, eventPeriod := range eventPeriods {
				eventLocationCompletionCategory.SubCategories = append(eventLocationCompletionCategory.SubCategories, common.RankingSubCategory{SubCategoryId: strconv.Itoa(eventPeriod.PeriodOrdinal), Game: gameName})
			}

			eventVmCountCategory := &common.RankingCategory{CategoryId: "eventVmCount", Game: gameName, Periodic: true}
			rankingCategories = append(rankingCategories, eventVmCountCategory)

			for _, eventPeriod := range eventPeriods {
				if eventPeriod.EnableVms {
					eventVmCountCategory.SubCategories = append(eventVmCountCategory.SubCategories, common.RankingSubCategory{SubCategoryId: strconv.Itoa(eventPeriod.PeriodOrdinal), Game: gameName})
				}
			}

			if len(eventVmCountCategory.SubCategories) > 1 {
				eventVmCountCategory.SubCategories = append([]common.RankingSubCategory{{SubCategoryId: "all", Game: gameName}}, eventVmCountCategory.SubCategories...)
			}
		}

		if gameName == "2kki" {
			timeTrialMapIds, err := database.GetTimeTrialMapIds()
			if err != nil {
				log.Print("SERVER", "timeTrial", err.Error())
			} else if len(timeTrialMapIds) > 0 {
				timeTrialCategory := &common.RankingCategory{CategoryId: "timeTrial", Game: gameName}
				rankingCategories = append(rankingCategories, timeTrialCategory)

				for _, mapId := range timeTrialMapIds {
					timeTrialCategory.SubCategories = append(timeTrialCategory.SubCategories, common.RankingSubCategory{SubCategoryId: strconv.Itoa(mapId), Game: gameName})
				}
			}
		}

		gameMinigameIds, err := database.GetGameMinigameIds(gameName)
		if err != nil {
			log.Print("SERVER", "minigame", err.Error())
		} else {
			minigameCategory := &common.RankingCategory{CategoryId: "minigame", Game: gameName}
			rankingCategories = append(rankingCategories, minigameCategory)

			for _, minigameId := range gameMinigameIds {
				minigameCategory.SubCategories = append(minigameCategory.SubCategories, common.RankingSubCategory{SubCategoryId: minigameId, Game: gameName})
			}
		}

		for c, category := range rankingCategories {
			err := database.WriteRankingCategory(category.CategoryId, category.Game, c)
			if err != nil {
				log.Print("SERVER", category.CategoryId, err.Error())
				continue
			}
			for sc, subCategory := range category.SubCategories {
				err = database.WriteRankingSubCategory(category.CategoryId, subCategory.SubCategoryId, subCategory.Game, sc)
				if err != nil {
					log.Print("SERVER", category.CategoryId+"/"+subCategory.SubCategoryId, err.Error())
				}
			}
		}

		common.GameRankingCategories[gameName] = rankingCategories
	}

	scheduler.Every(15).Minute().Do(func() {
		for _, gameName := range common.GameNames {
			for _, category := range common.GameRankingCategories[gameName] {
				for _, subCategory := range category.SubCategories {
					// Use Yume 2kki server to update 'all' rankings
					if subCategory.SubCategoryId == "all" && gameName != "2kki" {
						continue
					}
					if category.Periodic && subCategory.SubCategoryId != "all" {
						eventPeriodOrdinal, errconv := strconv.Atoi(subCategory.SubCategoryId)
						if errconv != nil || eventPeriodOrdinal != common.CurrentEventPeriodOrdinal {
							continue
						}
					}

					err := database.UpdateRankingEntries(category.CategoryId, subCategory.SubCategoryId)
					if err != nil {
						log.Print("SERVER", gameName+"/"+category.CategoryId+"/"+subCategory.SubCategoryId, err.Error())
					}
				}
			}

			err := database.UpdatePlayerMedals(gameName)
			if err != nil {
				log.Print("SERVER", "medals", err.Error())
			}
		}
	})

	scheduler.StartAsync()
}

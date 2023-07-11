package common

import (
	"time"
)

var (
	GameNames                 = []string{"2kki", "amillusion", "braingirl", "deepdreams", "flow", "mikan", "muma", "prayers", "someday", "ultraviolet", "unevendream", "yume"}
	GameRankingCategories     = make(map[string][]*RankingCategory)
	CurrentEventPeriodOrdinal = -1
)

type EventPeriod struct {
	PeriodOrdinal int       `json:"periodOrdinal"`
	EndDate       time.Time `json:"endDate"`
	EnableVms     bool      `json:"enableVms"`
}

type RankingCategory struct {
	CategoryId     string               `json:"categoryId"`
	Game           string               `json:"game"`
	SubCategories  []RankingSubCategory `json:"subCategories"`
	Periodic       bool                 `json:"periodic"`
	SeparateByGame bool                 `json:"-"`
}

type RankingSubCategory struct {
	SubCategoryId string `json:"subCategoryId"`
	Game          string `json:"game"`
	PageCount     int    `json:"pageCount"`
}

type Ranking struct {
	Position   int     `json:"position"`
	Name       string  `json:"name"`
	Rank       int     `json:"rank"`
	Badge      string  `json:"badge"`
	SystemName string  `json:"systemName"`
	Medals     [5]int  `json:"medals"`
	ValueInt   int     `json:"valueInt"`
	ValueFloat float32 `json:"valueFloat"`
}

type RankingEntry struct {
	CategoryId     string
	SubCategoryId  string
	Position       int
	ActualPosition int
	Uuid           string
	ValueInt       int
	ValueFloat     float32
	Timestamp      time.Time
}

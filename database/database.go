package database

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/ynoproject/ynorankings/common"

	_ "github.com/go-sql-driver/mysql"
)

var Conn *sql.DB

func Init() {
	conn, err := sql.Open("mysql", "yno@unix(/run/mysqld/mysqld.sock)/ynodb?parseTime=true")
	if err != nil {
		log.Fatal(err)
		return
	}

	Conn = conn
}

func GetPlayerUuidFromToken(token string) (uuid string) {
	err := Conn.QueryRow("SELECT a.uuid FROM accounts a JOIN playerSessions ps ON ps.uuid = a.uuid JOIN players pd ON pd.uuid = a.uuid WHERE ps.sessionId = ? AND NOW() < ps.expiration", token).Scan(&uuid)
	if err != nil {
		return ""
	}

	return uuid
}

func GetEventPeriodData(gameName string) (eventPeriods []*common.EventPeriod, err error) {
	results, err := Conn.Query("SELECT periodOrdinal, endDate, enableVms FROM eventPeriods WHERE game = ? AND periodOrdinal > 0", gameName)
	if err != nil {
		return eventPeriods, err
	}

	defer results.Close()

	for results.Next() {
		eventPeriod := &common.EventPeriod{}

		err := results.Scan(&eventPeriod.PeriodOrdinal, &eventPeriod.EndDate, &eventPeriod.EnableVms)
		if err != nil {
			return eventPeriods, err
		}

		eventPeriods = append(eventPeriods, eventPeriod)
	}

	return eventPeriods, nil
}

func GetCurrentEventPeriodId(gameName string) (periodId int, err error) {
	err = Conn.QueryRow("SELECT id FROM eventPeriods WHERE game = ? AND UTC_DATE() >= startDate AND UTC_DATE() < endDate", gameName).Scan(&periodId)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}

	return periodId, nil
}

func GetTimeTrialMapIds() (mapIds []int, err error) {
	results, err := Conn.Query("SELECT mapId FROM playerTimeTrials GROUP BY mapId ORDER BY MIN(seconds)")
	if err != nil {
		return mapIds, err
	}

	defer results.Close()

	for results.Next() {
		var mapId int
		err := results.Scan(&mapId)
		if err != nil {
			return mapIds, err
		}

		mapIds = append(mapIds, mapId)
	}

	return mapIds, nil
}

func GetGameMinigameIds(gameName string) (minigameIds []string, err error) {
	results, err := Conn.Query("SELECT DISTINCT minigameId FROM playerMinigameScores WHERE game = ? ORDER BY minigameId", gameName)
	if err != nil {
		return minigameIds, err
	}

	defer results.Close()

	for results.Next() {
		var minigameId string
		err := results.Scan(&minigameId)
		if err != nil {
			return minigameIds, err
		}

		minigameIds = append(minigameIds, minigameId)
	}

	return minigameIds, nil
}

func GetRankingCategories(gameName string) (rankingCategories []*common.RankingCategory, err error) {
	results, err := Conn.Query("SELECT categoryId, game FROM rankingCategories WHERE game IN ('', ?) ORDER BY ordinal", gameName)
	if err != nil {
		return rankingCategories, err
	}

	defer results.Close()

	for results.Next() {
		rankingCategory := &common.RankingCategory{}

		err := results.Scan(&rankingCategory.CategoryId, &rankingCategory.Game)
		if err != nil {
			return rankingCategories, err
		}

		rankingCategories = append(rankingCategories, rankingCategory)
	}

	results, err = Conn.Query("SELECT sc.categoryId, sc.subCategoryId, sc.game, CEILING(COUNT(r.uuid) / 25) FROM rankingSubCategories sc JOIN rankingEntries r ON r.categoryId = sc.categoryId AND r.subCategoryId = sc.subCategoryId WHERE sc.game IN ('', ?) GROUP BY sc.categoryId, sc.subCategoryId, sc.game ORDER BY 1, sc.ordinal", gameName)
	if err != nil {
		return rankingCategories, err
	}

	defer results.Close()

	var lastCategoryId string
	var lastCategory *common.RankingCategory

	for results.Next() {
		rankingSubCategory := &common.RankingSubCategory{}

		var categoryId string
		err := results.Scan(&categoryId, &rankingSubCategory.SubCategoryId, &rankingSubCategory.Game, &rankingSubCategory.PageCount)
		if err != nil {
			return rankingCategories, err
		}

		if lastCategoryId != categoryId {
			lastCategoryId = categoryId
			for _, rankingCategory := range rankingCategories {
				if rankingCategory.CategoryId == lastCategoryId {
					lastCategory = rankingCategory
				}
			}
		}

		lastCategory.SubCategories = append(lastCategory.SubCategories, *rankingSubCategory)
	}

	return rankingCategories, nil
}

func WriteRankingCategory(categoryId string, game string, order int) (err error) {
	_, err = Conn.Exec("INSERT INTO rankingCategories (categoryId, game, ordinal) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE ordinal = ?", categoryId, game, order, order)
	if err != nil {
		return err
	}

	return nil
}

func WriteRankingSubCategory(categoryId string, subCategoryId string, game string, order int) (err error) {
	_, err = Conn.Exec("INSERT INTO rankingSubCategories (categoryId, subCategoryId, game, ordinal) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE ordinal = ?", categoryId, subCategoryId, game, order, order)
	if err != nil {
		return err
	}

	return nil
}

func GetRankingEntryPage(playerUuid string, categoryId string, subCategoryId string) (page int, err error) {
	err = Conn.QueryRow("SELECT FLOOR(r.rowNum / 25) + 1 FROM (SELECT r.uuid, ROW_NUMBER() OVER (ORDER BY r.position) rowNum FROM rankingEntries r WHERE r.categoryId = ? AND r.subCategoryId = ? AND r.actualPosition <= 1000) r WHERE r.uuid = ?", categoryId, subCategoryId, playerUuid).Scan(&page)
	if err != nil {
		if err == sql.ErrNoRows {
			return 1, nil
		}
		return 1, err
	}

	return page, nil
}

func GetRankingsPaged(gameName string, categoryId string, subCategoryId string, page int) (rankings []*common.Ranking, err error) {
	var valueType string
	switch categoryId {
	case "eventLocationCompletion":
		valueType = "Float"
	default:
		valueType = "Int"
	}

	results, err := Conn.Query("SELECT r.position, a.user, pd.rank, a.badge, COALESCE(pgd.systemName, ''), r.value"+valueType+" FROM rankingEntries r JOIN accounts a ON a.uuid = r.uuid JOIN players pd ON pd.uuid = a.uuid LEFT JOIN playerGameData pgd ON pgd.uuid = pd.uuid AND pgd.game = ? WHERE r.categoryId = ? AND r.subCategoryId = ? ORDER BY r.actualPosition LIMIT "+strconv.Itoa((page-1)*25)+", 25", gameName, categoryId, subCategoryId)
	if err != nil {
		return rankings, err
	}

	defer results.Close()

	for results.Next() {
		ranking := &common.Ranking{}

		if valueType == "Int" {
			err = results.Scan(&ranking.Position, &ranking.Name, &ranking.Rank, &ranking.Badge, &ranking.SystemName, &ranking.ValueInt)
		} else {
			err = results.Scan(&ranking.Position, &ranking.Name, &ranking.Rank, &ranking.Badge, &ranking.SystemName, &ranking.ValueFloat)
		}
		if err != nil {
			return rankings, err
		}

		rankings = append(rankings, ranking)
	}

	return rankings, nil
}

func UpdateRankingEntries(categoryId string, subCategoryId string) (err error) {
	var valueType string
	var hasFloatValue bool
	switch categoryId {
	case "eventLocationCompletion":
		valueType = "Float"
		hasFloatValue = true
	default:
		valueType = "Int"
	}

	_, err = Conn.Exec("DELETE FROM rankingEntries WHERE categoryId = ? AND subCategoryId = ?", categoryId, subCategoryId)
	if err != nil {
		return err
	}

	isFiltered := subCategoryId != "all"
	var isUnion bool

	query := " "
	switch categoryId {
	case "badgeCount":
		query = "SELECT ?, ?, RANK() OVER (ORDER BY COUNT(pb.uuid) DESC), 0, a.uuid, COUNT(pb.uuid), (SELECT MAX(apb.timestampUnlocked) FROM playerBadges apb WHERE apb.uuid = a.uuid AND apb.badgeId = b.badgeId) FROM playerBadges pb JOIN accounts a ON a.uuid = pb.uuid JOIN badges b ON b.badgeId = pb.badgeId WHERE b.hidden = 0"
		if isFiltered {
			query += " AND b.game = ?"
		}
		query += " GROUP BY a.uuid"
	case "bp":
		query = "SELECT ?, ?, RANK() OVER (ORDER BY SUM(b.bp) DESC), 0, a.uuid, SUM(b.bp), (SELECT MAX(apb.timestampUnlocked) FROM playerBadges apb WHERE apb.uuid = a.uuid AND apb.badgeId = b.badgeId) FROM playerBadges pb JOIN accounts a ON a.uuid = pb.uuid JOIN badges b ON b.badgeId = pb.badgeId"
		if isFiltered {
			query += " WHERE b.game = ?"
		}
		query += " GROUP BY a.uuid"
	case "exp":
		query = "SELECT ?, ?, RANK() OVER (ORDER BY SUM(ec.exp) DESC), 0, ec.uuid, SUM(ec.exp), (SELECT MAX(aec.timestampCompleted) FROM eventCompletions aec WHERE aec.uuid = ec.uuid) FROM ((SELECT ec.uuid, ec.exp FROM eventCompletions ec JOIN eventLocations el ON el.id = ec.eventId AND ec.type = 0"
		if isFiltered {
			query += " JOIN eventPeriods ep ON ep.id = el.periodId AND ep.periodOrdinal = ?"
		}
		query += ") UNION ALL (SELECT ec.uuid, ec.exp FROM eventCompletions ec JOIN eventVms ev ON ev.id = ec.eventId AND ec.type = 2"
		if isFiltered {
			query += " JOIN eventPeriods ep ON ep.id = ev.periodId AND ep.periodOrdinal = ?"
		}
		query += ")) ec GROUP BY ec.uuid"
		isUnion = true
	case "eventLocationCount", "freeEventLocationCount":
		isFree := categoryId == "freeEventLocationCount"
		query = "SELECT ?, ?, RANK() OVER (ORDER BY COUNT(ec.uuid) DESC), 0, ec.uuid, COUNT(ec.uuid), (SELECT MAX(aec.timestampCompleted) FROM eventCompletions aec WHERE aec.uuid = ec.uuid) FROM eventCompletions ec "
		if isFiltered {
			if isFree {
				query += "JOIN playerEventLocations el"
			} else {
				query += "JOIN eventLocations el"
			}
			query += " ON el.id = ec.eventId JOIN eventPeriods ep ON ep.id = el.periodId AND ep.periodOrdinal = ? "
		}
		query += "WHERE ec.type = "
		if isFree {
			query += "1"
		} else {
			query += "0"
		}
		query += " GROUP BY ec.uuid"
	case "eventLocationCompletion":
		query = "SELECT ?, ?, RANK() OVER (ORDER BY COUNT(DISTINCT COALESCE(el.title, pel.title)) / aec.count DESC), 0, a.uuid, COUNT(DISTINCT COALESCE(el.title, pel.title)) / aec.count, (SELECT MAX(aect.timestampCompleted) FROM eventCompletions aect WHERE aect.uuid = ec.uuid) FROM eventCompletions ec JOIN accounts a ON a.uuid = ec.uuid LEFT JOIN eventLocations el ON el.id = ec.eventId AND ec.type = 0 LEFT JOIN playerEventLocations pel ON pel.id = ec.eventId AND ec.type = 1 JOIN (SELECT COUNT(DISTINCT COALESCE(ael.title, apel.title)) count FROM eventCompletions aec LEFT JOIN eventLocations ael ON ael.id = aec.eventId AND aec.type = 0 LEFT JOIN playerEventLocations apel ON apel.id = aec.eventId AND aec.type = 1 WHERE (ael.title IS NOT NULL OR apel.title IS NOT NULL)) aec"
		if isFiltered {
			query += " JOIN eventPeriods ep ON ep.id = COALESCE(el.periodId, pel.periodId) AND ep.periodOrdinal = ?"
		}
		query += " GROUP BY a.user"
	case "eventVmCount":
		query = "SELECT ?, ?, RANK() OVER (ORDER BY COUNT(ec.uuid) DESC), 0, ec.uuid, COUNT(ec.uuid), (SELECT MAX(aec.timestampCompleted) FROM eventCompletions aec WHERE aec.uuid = ec.uuid) FROM eventCompletions ec "
		if isFiltered {
			query += "JOIN eventVms ev ON ev.id = ec.eventId JOIN eventPeriods ep ON ep.id = ev.periodId AND ep.periodOrdinal = ? "
		}
		query += "WHERE ec.type = 2 GROUP BY ec.uuid"
	case "timeTrial":
		query = "SELECT ?, ?, RANK() OVER (ORDER BY MIN(tt.seconds)), 0, tt.uuid, MIN(tt.seconds), (SELECT MAX(att.timestampCompleted) FROM playerTimeTrials att WHERE att.uuid = tt.uuid AND att.mapId = tt.mapId AND att.seconds = tt.seconds) FROM playerTimeTrials tt WHERE tt.mapId = ? GROUP BY tt.uuid"
	case "minigame":
		query = "SELECT ?, ?, RANK() OVER (ORDER BY MAX(ms.score) DESC), 0, ms.uuid, MAX(ms.score), (SELECT MAX(ams.timestampCompleted) FROM playerMinigameScores ams WHERE ams.uuid = ms.uuid AND ams.minigameId = ms.minigameId AND ams.score = ms.score) FROM playerMinigameScores ms WHERE ms.minigameId = ? GROUP BY ms.uuid"
	}

	query += " ORDER BY 3, 6"

	var results *sql.Rows
	if isFiltered {
		if isUnion {
			results, err = Conn.Query(query, categoryId, subCategoryId, subCategoryId, subCategoryId)
		} else {
			results, err = Conn.Query(query, categoryId, subCategoryId, subCategoryId)
		}
	} else {
		results, err = Conn.Query(query, categoryId, subCategoryId)
	}
	if err != nil {
		return err
	}

	defer results.Close()

	var placeholders []string
	var entryValues []interface{}
	var rowIndex int

	for results.Next() {
		placeholders = append(placeholders, "(?, ?, ?, ?, ?, ?, ?)")

		entry := &common.RankingEntry{}
		if hasFloatValue {
			results.Scan(&entry.CategoryId, &entry.SubCategoryId, &entry.Position, &entry.ActualPosition, &entry.Uuid, &entry.ValueFloat, &entry.Timestamp)
		} else {
			results.Scan(&entry.CategoryId, &entry.SubCategoryId, &entry.Position, &entry.ActualPosition, &entry.Uuid, &entry.ValueInt, &entry.Timestamp)
		}
		entryValues = append(entryValues, entry.CategoryId, entry.SubCategoryId, entry.Position, entry.ActualPosition, entry.Uuid)
		if hasFloatValue {
			entryValues = append(entryValues, entry.ValueFloat)
		} else {
			entryValues = append(entryValues, entry.ValueInt)
		}
		entryValues = append(entryValues, entry.Timestamp)

		if rowIndex == 1000 {
			break
		}

		rowIndex++
	}

	if len(entryValues) == 0 {
		return nil
	}

	insertQuery := fmt.Sprintf("INSERT INTO rankingEntries (categoryId, subCategoryId, position, actualPosition, uuid, value"+valueType+", timestamp) VALUES %s", strings.Join(placeholders, ","))
	_, err = Conn.Exec(insertQuery, entryValues...)
	if err != nil {
		return err
	}

	_, err = Conn.Exec("UPDATE rankingEntries e JOIN (WITH re AS (SELECT e.categoryId, e.subCategoryId, e.position, e.timestamp, ROW_NUMBER() OVER (ORDER BY e.position, e.timestamp) actualPosition FROM rankingEntries e WHERE e.categoryId = ? AND e.subCategoryId = ?) SELECT * FROM re) re ON re.categoryId = e.categoryId AND re.subCategoryId = e.subCategoryId AND re.position = e.position AND re.timestamp = e.timestamp SET e.actualPosition = re.actualPosition", categoryId, subCategoryId)
	if err != nil {
		return err
	}

	return nil
}

func UpdatePlayerMedals(gameName string) (err error) {
	_, err = Conn.Exec("UPDATE playerGameData pgd JOIN (SELECT uuid, SUM(CASE WHEN actualPosition <= 100 AND actualPosition > 30 THEN 1 ELSE 0 END) bronze, SUM(CASE WHEN actualPosition <= 30 AND actualPosition > 10 THEN 1 ELSE 0 END) silver, SUM(CASE WHEN actualPosition <= 10 AND actualPosition > 1 THEN 1 ELSE 0 END) gold, SUM(CASE WHEN actualPosition <= 3 AND actualPosition > 1 THEN 1 ELSE 0 END) plat, SUM(CASE WHEN actualPosition = 1 THEN 1 ELSE 0 END) diamond FROM rankingEntries e JOIN rankingCategories rc ON rc.categoryId = e.categoryId JOIN rankingSubCategories rsc ON rsc.categoryId = e.categoryId AND rsc.subCategoryId = e.subCategoryId AND rc.game IN ('', ?) AND rsc.game IN ('', ?) WHERE (rc.periodic = 0 OR e.subCategoryId IN ('all', ?)) GROUP BY uuid) m ON m.uuid = pgd.uuid SET pgd.medalCountBronze = m.bronze, pgd.medalCountSilver = m.silver, pgd.medalCountGold = m.gold, pgd.medalCountPlatinum = m.plat, pgd.medalCountDiamond = m.diamond WHERE pgd.game = ?", gameName, gameName, common.GameCurrentEventPeriodIds[gameName], gameName)
	if err != nil {
		return err
	}

	return nil
}

package api

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"ynorankings/database"
)

func Init() {
	http.HandleFunc("/", handleRankings)

	http.Serve(getListener(), nil)
}

func getListener() net.Listener {
	os.Remove("sockets/rankings.sock")

	listener, err := net.Listen("unix", "sockets/rankings.sock")
	if err != nil {
		log.Fatal(err)
		return nil
	}

	if err := os.Chmod("sockets/rankings.sock", 0666); err != nil {
		log.Fatal(err)
		return nil
	}

	return listener
}

func handleRankings(w http.ResponseWriter, r *http.Request) {
	var uuid string

	token := r.Header.Get("Authorization")
	if token != "" {
		uuid = database.GetPlayerUuidFromToken(token)
	}

	gameParam, ok := r.URL.Query()["game"]
	if !ok || len(gameParam) == 0 {
		http.Error(w, "game not specified", http.StatusBadRequest)
		return
	}

	commandParam, ok := r.URL.Query()["command"]
	if !ok || len(commandParam) == 0 {
		http.Error(w, "command not specified", http.StatusBadRequest)
		return
	}

	switch commandParam[0] {
	case "categories":
		rankingCategories, err := database.GetRankingCategories(gameParam[0])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		rankingCategoriesJson, err := json.Marshal(rankingCategories)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(rankingCategoriesJson)
	case "page":
		categoryParam, ok := r.URL.Query()["category"]
		if !ok || len(categoryParam) == 0 {
			http.Error(w, "category not specified", http.StatusBadRequest)
			return
		}

		subCategoryParam, ok := r.URL.Query()["subCategory"]
		if !ok || len(subCategoryParam) == 0 {
			http.Error(w, "subcategory not specified", http.StatusBadRequest)
			return
		}

		playerPage := 1
		if token != "" {
			var err error
			playerPage, err = database.GetRankingEntryPage(uuid, categoryParam[0], subCategoryParam[0])
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		w.Write([]byte(strconv.Itoa(playerPage)))
	case "list":
		categoryParam, ok := r.URL.Query()["category"]
		if !ok || len(categoryParam) == 0 {
			http.Error(w, "category not specified", http.StatusBadRequest)
			return
		}

		subCategoryParam, ok := r.URL.Query()["subCategory"]
		if !ok || len(subCategoryParam) == 0 {
			http.Error(w, "subcategory not specified", http.StatusBadRequest)
			return
		}

		var page int
		pageParam, ok := r.URL.Query()["page"]
		if !ok || len(pageParam) == 0 {
			page = 1
		} else {
			pageInt, err := strconv.Atoi(pageParam[0])
			if err != nil {
				page = 1
			} else {
				page = pageInt
			}
		}

		rankings, err := database.GetRankingsPaged(gameParam[0], categoryParam[0], subCategoryParam[0], page)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		rankingsJson, err := json.Marshal(rankings)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(rankingsJson)
	default:
		http.Error(w, "unknown command", http.StatusBadRequest)
	}
}

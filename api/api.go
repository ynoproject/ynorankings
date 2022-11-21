package api

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/ynoproject/ynorankings/database"
)

func Init() {
	http.HandleFunc("/categories", handleCategories)
	http.HandleFunc("/page", handlePage)
	http.HandleFunc("/list", handleList)

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

func handleCategories(w http.ResponseWriter, r *http.Request) {
	gameParam, ok := r.URL.Query()["game"]
	if !ok || len(gameParam) == 0 {
		http.Error(w, "game not specified", http.StatusBadRequest)
		return
	}

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
}

func handlePage(w http.ResponseWriter, r *http.Request) {
	var uuid string

	token := r.Header.Get("Authorization")
	if token != "" {
		uuid = database.GetPlayerUuidFromToken(token)
	}

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
}

func handleList(w http.ResponseWriter, r *http.Request) {
	gameParam, ok := r.URL.Query()["game"]
	if !ok || len(gameParam) == 0 {
		http.Error(w, "game not specified", http.StatusBadRequest)
		return
	}

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
}

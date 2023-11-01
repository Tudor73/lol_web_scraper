package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"github.com/gorilla/mux"
)

type Champion struct {
	Name     string
	WinRate  float32
	PlayRate string
	Counters []Champion
}

type Result struct {
	Counters []Champion `json:"counters"`
	Result   string     `json:"result"`
}

func CreateURL(championName string) string {
	url := url.URL{
		Scheme: "https",
		Host:   "u.gg",
		Path:   fmt.Sprintf("/lol/champions/%s/counter", strings.Trim(championName, " \n")),
	}
	return url.String()
}

func main() {

	r := mux.NewRouter()
	r.HandleFunc("/", handler).Methods("GET")
	r.HandleFunc("/counters/{championName}", championHandler).Methods("GET")

	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("server running"))
}

func championHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	championName := vars["championName"]

	c := colly.NewCollector()
	var wg sync.WaitGroup
	var mutex sync.Mutex

	mainChampion := Champion{championName, 0, "main", make([]Champion, 0)}
	c.OnError(func(_ *colly.Response, err error) {
		fmt.Println("Something went wrong: ", err)
		http.Error(w, err.Error(), 400)
	})
	c.OnResponse(func(r *colly.Response) {
		fmt.Println("Page visited: ", r.Request.URL)
	})
	c.OnHTML(".best-win-rate", getHTMLCallback(&mainChampion, &mutex))
	url := CreateURL(mainChampion.Name)
	err := c.Visit(url)
	if err != nil {
		http.Error(w, err.Error(), 400)
	}

	for _, c := range mainChampion.Counters {
		fmt.Println(c.Name, c.WinRate, "%", c.PlayRate)
	}
	fmt.Println()
	fmt.Println("Finding best champion for pool...")
	for idx := range mainChampion.Counters {
		wg.Add(1)
		var champ = &mainChampion.Counters[idx]

		go func(wg *sync.WaitGroup, champ *Champion) {
			url := CreateURL(champ.Name)
			c := colly.NewCollector()
			c.OnHTML(".best-win-rate", getHTMLCallback(champ, &mutex))
			c.OnScraped(func(r *colly.Response) {
				wg.Done()
			})
			c.OnError(func(r *colly.Response, err error) {
				fmt.Println("error when scraping site", err)
				http.Error(w, err.Error(), 500)
				wg.Done()

			})
			c.Visit(url)
			// defer wg.Done()
		}(&wg, champ)

	}

	wg.Wait()
	var result Result
	for _, champ := range mainChampion.Counters {
		fmt.Println(champ.Name)
		result.Counters = append(result.Counters, champ)
		for _, c := range champ.Counters {
			fmt.Println(c.Name, c.WinRate, c.PlayRate)
		}
		fmt.Println()
	}
	champToPlay := pickChampToPlay(mainChampion)
	result.Result = champToPlay
	fmt.Println("Best champion to add is", champToPlay)

	jsonStr, err := json.Marshal(result)

	if err != nil {
		http.Error(w, err.Error(), 500)
	}
	w.Write(jsonStr)
}

func getHTMLCallback(c *Champion, mutex *sync.Mutex) colly.HTMLCallback {
	return func(e *colly.HTMLElement) {
		name := e.ChildText(".champion-name")
		if len(name) > 15 {
			return
		}
		winRate, err := strconv.ParseFloat(strings.Trim(e.ChildText(".win-rate"), "% WR"), 32)

		if winRate < 50 {
			return
		}
		if err != nil {
			log.Panic("err when parsing win rate", err)
		}
		playRate := e.ChildText(".total-games")

		newChamp := Champion{name, float32(winRate), playRate, make([]Champion, 0)}

		// mutex.Lock()
		c.Counters = append(c.Counters, newChamp)
		// mutex.Unlock()
	}
}

func pickChampToPlay(mainChampion Champion) string {
	hashMap := make(map[string]int)
	maxim, result := 0, ""

	for _, champ := range mainChampion.Counters {
		for _, c := range champ.Counters {
			v, ok := hashMap[c.Name]
			if ok {
				v++
				hashMap[c.Name] = v
			} else {
				hashMap[c.Name] = 1
			}
			if hashMap[c.Name] > maxim {
				maxim = hashMap[c.Name]
				result = c.Name
			}
		}
	}
	return result
}

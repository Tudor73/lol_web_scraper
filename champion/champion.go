package champion

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
	"github.com/labstack/echo/v4"
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
func ChampionHandler(context echo.Context) error {
	championName := context.Param("championName")

	c := colly.NewCollector()
	var wg sync.WaitGroup

	mainChampion := Champion{championName, 0, "main", make([]Champion, 0)}
	c.OnError(func(_ *colly.Response, err error) {
		fmt.Println("Something went wrong: ", err)
		return
	})
	c.OnResponse(func(r *colly.Response) {
		fmt.Println("Page visited: ", r.Request.URL)
	})
	c.OnHTML(".best-win-rate", getHTMLCallback(&mainChampion))
	url := CreateURL(mainChampion.Name)
	err := c.Visit(url)
	if err != nil {
		return err
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
			c.OnHTML(".best-win-rate", getHTMLCallback(champ))
			c.OnScraped(func(r *colly.Response) {
				wg.Done()
			})
			c.OnError(func(r *colly.Response, err error) {
				fmt.Println("error when scraping site", err)
				wg.Done()
				return

			})
			c.Visit(url)
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

	context.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	context.Response().WriteHeader(http.StatusOK)
	return json.NewEncoder(context.Response()).Encode(result)
}

func getHTMLCallback(c *Champion) colly.HTMLCallback {
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
		c.Counters = append(c.Counters, newChamp)
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

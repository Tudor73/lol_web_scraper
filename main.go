package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gocolly/colly"
)

type Champion struct {
	Name     string
	WinRate  float32
	PlayRate string
	Counters []Champion
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

	c := colly.NewCollector()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter the champion name: ")

	championName, err := reader.ReadString('\n')
	mainChampion := Champion{championName, 0, "main", make([]Champion, 0)}
	if err != nil {
		log.Panic("error when reading champion name: ", err)
	}
	c.OnError(func(_ *colly.Response, err error) {
		fmt.Println("Something went wrong: ", err)
	})
	c.OnResponse(func(r *colly.Response) {
		fmt.Println("Page visited: ", r.Request.URL)
	})
	c.OnHTML(".best-win-rate", func(e *colly.HTMLElement) {
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
		mainChampion.Counters = append(mainChampion.Counters, newChamp)

	})
	url := CreateURL(championName)
	c.Visit(url)

	for _, c := range mainChampion.Counters {
		fmt.Println(c.Name, c.WinRate, "%", c.PlayRate)
	}
	fmt.Println()
	fmt.Println("Finding best champion for pool...")

	for idx := range mainChampion.Counters {
		url = CreateURL(mainChampion.Counters[idx].Name)
		c := colly.NewCollector()
		c.OnHTML(".best-win-rate", getHTMLCallback(&mainChampion.Counters[idx]))
		c.Visit(url)
	}

	for _, champ := range mainChampion.Counters {
		fmt.Println(champ.Name)
		for _, c := range champ.Counters {
			fmt.Println(c.Name, c.WinRate, c.PlayRate)
		}
		fmt.Println()
	}

	champToPlay := pickChampToPlay(mainChampion)
	fmt.Println("Best champion to add is ", champToPlay)
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

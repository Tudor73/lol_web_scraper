package main

import (
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"sync"

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
	var wg sync.WaitGroup
	var mutex sync.Mutex
	// reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter the champion name: ")

	// championName, err := reader.ReadString('\n')
	mainChampion := Champion{"garen", 0, "main", make([]Champion, 0)}
	// if err != nil {
	// 	log.Panic("error when reading champion name: ", err)
	// }
	c.OnError(func(_ *colly.Response, err error) {
		fmt.Println("Something went wrong: ", err)
	})
	c.OnResponse(func(r *colly.Response) {
		fmt.Println("Page visited: ", r.Request.URL)
	})
	c.OnHTML(".best-win-rate", getHTMLCallback(&mainChampion, &mutex))
	url := CreateURL(mainChampion.Name)
	c.Visit(url)

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
				wg.Done()

			})
			c.Visit(url)
			// defer wg.Done()
		}(&wg, champ)

	}

	wg.Wait()
	for _, champ := range mainChampion.Counters {
		fmt.Println(champ.Name)
		for _, c := range champ.Counters {
			fmt.Println(c.Name, c.WinRate, c.PlayRate)
		}
		fmt.Println()
	}

	champToPlay := pickChampToPlay(mainChampion)
	fmt.Println("Best champion to add is", champToPlay)
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

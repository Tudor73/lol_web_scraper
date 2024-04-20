package main

import (
	"lol_web_scraper/champion"
	"net/http"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()
	e.GET("/", handler)
	e.GET("/counters/:championName", champion.ChampionHandler)
	e.Logger.Fatal(e.Start(":8080"))

}

func handler(c echo.Context) error {
	return c.String(http.StatusOK, "Server runnning test!")
}

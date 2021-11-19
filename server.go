package main

import (
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
	"node-selector/node/kda"
	"os"
	"path/filepath"
)

type Config struct {
	Nodes []struct {
		Coin   string   `json:"coin"`
		Domain int      `json:"domain"`
		Urls   []string `json:"urls"`
	} `json:"nodes"`
}

var cfg Config

func main() {
	readConfig(&cfg)
	var s kda.Selector
	for _, node := range cfg.Nodes {
		if node.Coin == "kda" {
			s.Selector(node.Urls, node.Domain)
		}
	}
	e := echo.New()
	e.GET("/", func(context echo.Context) error {
		return context.String(http.StatusOK, "node-selector (* ￣︿￣)")
	})
	e.GET("/node/:coin", getNode)
	e.Logger.Fatal(e.Start(":9826"))
}

func readConfig(cfg *Config) {
	configFileName := "config.json"
	if len(os.Args) > 1 {
		configFileName = os.Args[1]
	}
	configFileName, _ = filepath.Abs(configFileName)
	log.Printf("Loading config: %v", configFileName)

	configFile, err := os.Open(configFileName)
	if err != nil {
		log.Fatal("File error: ", err.Error())
	}
	defer configFile.Close()
	jsonParser := json.NewDecoder(configFile)
	if err := jsonParser.Decode(&cfg); err != nil {
		log.Fatal("Config error: ", err.Error())
	}
}

func getNode(c echo.Context) error {
	coin := c.Param("coin")
	conn, err := redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		fmt.Println("redis.Dial err=", err)
		return c.String(http.StatusOK, "failed to connect redis")
	}
	defer conn.Close()

	node, err := redis.String(conn.Do("GET", "node:"+coin))
	if err != nil {
		fmt.Println("redis.Dial err=", err)
		return c.String(http.StatusOK, "failed to get kda node")
	}
	fmt.Println("node:", node)
	return c.String(http.StatusOK, node)
}

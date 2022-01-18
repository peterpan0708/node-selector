package main

import "node-selector/api"

type Config struct {
    Nodes []struct {
        Coin   string   `json:"coin"`
        Domain int      `json:"domain"`
        Urls   []string `json:"urls"`
    } `json:"nodes"`
}


func main() {
    var a api.Api
    a.CreateServer()
}
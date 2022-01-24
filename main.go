package main

import (
    "fmt"
    "github.com/jinzhu/configor"
    "node-selector/api"
    "node-selector/cmd/node/kda"
    "node-selector/configs"
    "node-selector/tools/system"
)

func main() {
    var cfg configs.Config
    var a api.Api
    err := configor.Load(&cfg, "configs/config.json")
    if err != nil {
        fmt.Println("read config err=", err)
        return
    }
    chanData := make(chan interface{})
    go a.CreateShortServer(cfg.Api)
    go a.CreatePersistentServer(cfg.Api, chanData)
    for _, c := range cfg.Nodes {
        if c.Name == "kda" {
            var k kda.Node
            go k.Start(c, chanData)
        }
    }
    system.Quit()
}

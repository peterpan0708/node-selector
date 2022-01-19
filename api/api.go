package api

import (
    "fmt"
    "github.com/labstack/echo/v4"
    "gopkg.in/redis.v5"
    "net/http"
    "node-selector/configs"
    redisOperation "node-selector/tools/redis"
)
type Api struct {
    RedisClient *redis.Client
}

type Config struct {
    Nodes []struct {
        Coin   string   `json:"coin"`
        Domain int      `json:"domain"`
        Urls   []string `json:"urls"`
    } `json:"nodes"`
}




func (a *Api) CreateServer(api configs.Api) {
    //client := redis.NewClient(&redis.Options{
    //    Addr:     "127.0.0.1:6379",
    //    Password: "", // no password set
    //    DB:       0,  // use default DB
    //})
    //a.RedisClient =client
    //
    //url := api.Host+":"+string(rune(api.Port))
    //netListen, err := net.Listen("tcp", url)
    //if err != nil {
    //    fmt.Println("create server err=", err)
    //    return
    //}
    //defer netListen.Close()
    //
    ////等待客户端访问
    //for {
    //    //监听接收
    //    conn, err := netListen.Accept()
    //    //如果发生错误，继续下一个循环。
    //    if err != nil {
    //        continue
    //    }
    //    fmt.Println("tcp connect success")
    //
    //    // 模拟汇报挖空块
    //    a.HandleConnection(conn, ltcData)
    //
    //}
}

func (a *Api) getNode(c echo.Context) error {
    coin := c.Param("coin")
    //conn, err := redis.Dial("tcp", "127.0.0.1:6379")
    //if err != nil {
    //    fmt.Println("redis.Dial err=", err)
    //    return c.String(http.StatusOK, "failed to connect redis")
    //}
    //defer conn.Close()

    node, err := redisOperation.Get(a.RedisClient, "node:"+coin).Result()
    if err != nil {
        fmt.Println("redis.Dial err=", err)
        return c.String(http.StatusOK, "failed to get kda node")
    }
    fmt.Println("node:", node)
    return c.String(http.StatusOK, node)
}

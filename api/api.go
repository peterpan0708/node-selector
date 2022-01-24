package api

import (
    "fmt"
    "github.com/labstack/echo/v4"
    "gopkg.in/redis.v5"
    "net"
    "net/http"
    "node-selector/configs"
    redisOperation "node-selector/tools/redis"
)

type Api struct {
    RedisClient *redis.Client
}

func (a *Api) CreateShortServer(cfg configs.Api) {
    client := redis.NewClient(&redis.Options{
        Addr:     "127.0.0.1:6379",
        Password: "", // no password set
        DB:       0,  // use default DB
    })
    a.RedisClient = client
    e := echo.New()
    e.GET("/", func(context echo.Context) error {
        return context.String(http.StatusOK, "node-selector (* ￣︿￣)")
    })
    e.GET("/node/:coin", a.getNode)
    e.Logger.Fatal(e.Start(":9826"))
}

func (a *Api) CreatePersistentServer(api configs.Api, chanData chan interface{}) {
    client := redis.NewClient(&redis.Options{
        Addr:     "127.0.0.1:6379",
        Password: "", // no password set
        DB:       0,  // use default DB
    })
    a.RedisClient = client

    url := api.Host + ":" + string(rune(api.Port))
    netListen, err := net.Listen("tcp", url)
    if err != nil {
        fmt.Println("create server err=", err)
        return
    }
    defer netListen.Close()

    //等待客户端访问
    for {
        //监听接收
        conn, err := netListen.Accept()
        //如果发生错误，继续下一个循环。
        if err != nil {
            continue
        }
        fmt.Println("tcp connect success")

        // 模拟汇报挖空块
        a.HandleConnection(conn, chanData)

    }
}

func (a *Api) getNode(c echo.Context) error {
    coin := c.Param("coin")

    node, err := redisOperation.Get(a.RedisClient, coin+":fastestNode").Result()
    if err != nil {
        fmt.Println("redis.Dial err=", err)
        return c.String(http.StatusOK, "failed to get kda node")
    }
    fmt.Println("node:", node)
    return c.String(http.StatusOK, node)
}

func (a *Api) HandleConnection(conn net.Conn, chanData chan interface{}) {
    buffer := make([]byte, 2048) //建立一个slice
    for {
        //读取客户端传来的内容
        n, err := conn.Read(buffer)
        // 当远程客户端连接发生错误（断开）后，终止此协程。
        if err != nil {
            fmt.Println("connection error: ", err)
            return
        }
        fmt.Println("receive data string:\n", string(buffer[:n]))

        if string(buffer[:n]) != "startPoolWatcher" {
            continue
        }

        //返回给客户端的信息
        for data := range chanData {
            fmt.Println("SEND:", data)
            str := fmt.Sprintf("%v", data)
            _, err := conn.Write([]byte(str))
            if err != nil {
                fmt.Println("tell err:", err.Error())
            }
        }
    }
}

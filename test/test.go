package main

import (
    "bytes"
    "crypto/tls"
    "encoding/json"
    "fmt"
    "github.com/asmcos/requests"
    "gopkg.in/redis.v5"
    "io"
    "log"
    "net/http"
    redisOperation "node-selector/tools/redis"
    "strconv"
    "time"
)

// 本质上是做了一个加权评分系统

func main() {
    healthCheck := make(chan interface{})
    go func() {
        for x := range healthCheck {
            fmt.Println(x)
            if x == 1001 {
                fmt.Println("GetNodeHeight健康")
            }
            if x == 1002 {
                fmt.Println("GetMiningWork健康")
            }
            if x == 1003 {
                fmt.Println("GetUpdatesWithFunc健康")
            }
            if x == 0 {
                fmt.Println("不健康")
            }
        }
    }()

    // 建立1848 updates 长连接
    GetUpdatesWithFunc("47.101.48.191", HandleConnection, healthCheck)
    //fmt.Println(g)
    //fmt.Println(c)
    //fmt.Println(t)

    // 1848 一次性get work
    GetMiningWork("47.101.48.191", healthCheck)

    //444 获取高度
    GetNodeHeight("47.101.48.191", healthCheck)
    //if err != nil {
    //    fmt.Println("failed")
    //} else {
    //    fmt.Println("高度:", h)
    //}

    time.Sleep(100 * time.Second)
}

//// isHeightBlocked 高度是否停滞,如果停滞,healthCheck输出-1,当main里检测到-1之后,健康得分-5
//func isHeightBlocked(url string, healthCheck chan interface{}) {
//    client := redis.NewClient(&redis.Options{
//        Addr:     "127.0.0.1:6379",
//        Password: "", // no password set
//        DB:       0,  // use default DB
//    })
//    RedisClient := client
//
//    keys, err := redisOperation.Keys(RedisClient, "kda:47.101.48.191").Result()
//    if err != nil {
//        fmt.Println("redis get kda data error=", err.Error())
//        return
//    }
//    for _, v := range keys {
//        timeStamp, err := redisOperation.Get(RedisClient, v).Result()
//        t, err := strconv.ParseInt(timeStamp, 10, 64)
//        if err != nil {
//            fmt.Println("strconv.ParseInt err=", err)
//            return
//        }
//
//    }
//
//}

type CutResp struct {
    Hashes map[string]struct {
        Height int64 `json:"height"`
    } `json:"hashes"`
}

func GetNodeHeight(url string, healthCheck chan interface{}) {
    duration := time.Second * 1
    timer := time.NewTimer(duration)

    go func() {
        for {
            select {
            case <-timer.C:

                //retry := 1
                var res CutResp
                req := requests.Requests()
                tr := &http.Transport{
                    TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
                }
                req.Client.Transport = tr
                req.SetTimeout(5)
                //for retry > 0 {

                resp, err := req.Get("https://" + url + ":444/chainweb/0.0/mainnet01/cut")
                if err != nil {
                    fmt.Printf("failed to get %s block height, err: %v \n", url, err.Error())
                    //retry--
                } else {
                    err = resp.Json(&res)
                    if err != nil {
                        fmt.Printf("failed to unmarshal %s block height err: %v \n", url, err.Error())
                        healthCheck <- 0
                    } else {
                        height := res.Hashes["0"].Height
                        if height == 0 {
                            healthCheck <- 0
                        }
                        healthCheck <- 1001
                    }
                }
                //err = errors.New("failed to get " + url + " block height after 3 times")
                //println("failed to get " + url + " block height after 3 times")
                timer.Reset(duration)
            }
        }
    }()
}

// 1848 一次性
func GetMiningWork(url string, healthCheck chan interface{}) {
    duration := time.Second * 1
    timer := time.NewTimer(duration)

    go func() {
        for {
            select {
            case <-timer.C:

                body := "{\n    \"account\": \"96569b08da3a631b1ca7f2cc768f2f14723510ace3b36b36f3be07f233d65596\",\n    \"predicate\": \"keys-all\",\n    \"public-keys\": [\n        \"96569b08da3a631b1ca7f2cc768f2f14723510ace3b36b36f3be07f233d65596\"\n    ]\n}"
                strJson := []byte(body)
                buffJson := bytes.NewBuffer(strJson)
                request, err := http.NewRequest("GET", "http://"+url+":1848/chainweb/0.0/mainnet01/mining/work?chain=0", buffJson)
                request.Header.Add("Connection", "keep-alive")
                request.Header.Add("Content-Type", "application/json;charset=utf-8")
                request.Header.Add("Transfer-Encoding", "chunked")
                if err != nil {
                    log.Fatal(err)
                }
                http_client := &http.Client{}
                response, err := http_client.Do(request)
                if err != nil {
                    fmt.Printf("An error occurred in the Node:%s, error is %s \n", url, err)
                    healthCheck <- 0
                    //log.Fatal(err)
                    //return
                }
                defer response.Body.Close()

                buf := make([]byte, 1024)
                n, err := response.Body.Read(buf)
                if n == 0 && err != nil {
                    fmt.Printf("An error occurred in the Node:%s, error is %s \n", url, err)
                    healthCheck <- 0
                    //log.Fatal(err)
                    //return
                }
                //fmt.Println(string(buf[:n]))
                //fmt.Println(string(buf))
                //

                //s := hex.EncodeToString(buf[:n])
                //fmt.Println(s)
                healthCheck <- 1002
                timer.Reset(duration)
            }
        }
    }()
}

// 1848 长连接 存入redis内,如果多久没推,就判定有问题
func GetUpdatesWithFunc(url string, handleConnection func(conn io.ReadCloser, healthCheck chan interface{}), healthCheck chan interface{}) {
    request, err := http.NewRequest("GET", "http://"+url+":1848/chainweb/0.0/mainnet01/header/updates", nil)
    request.Header.Add("Connection", "keep-alive")
    request.Header.Add("Pragma", "no-cache")
    request.Header.Add("Cache-Control", "no-cache")
    request.Header.Add("Accept", "text/event-stream")
    request.Header.Add("Sec-Fetch-Site", "same-site")
    request.Header.Add("Sec-Fetch-Mode", "cors")
    request.Header.Add("Sec-Fetch-Dest", "empty")
    request.Header.Add("Referer", "https://explorer.chainweb.com/mainnet")
    request.Header.Add("Accept-Language", "zh-CN,zh;q=0.9")
    request.Header.Add("Transfer-Encoding", "chunked")

    if err != nil {
        log.Fatal(err)
    }

    http_client := &http.Client{}
    response, err := http_client.Do(request)
    if err != nil {
        fmt.Printf("An error occurred in the Node:%s, error is %s \n", url, err)
        healthCheck <- 0
        //log.Fatal(err)
    }
    handleConnection(response.Body, healthCheck)
    return
}

func HandleConnection(conn io.ReadCloser, healthCheck chan interface{}) {
    go func() {
        client := redis.NewClient(&redis.Options{
            Addr:     "127.0.0.1:6379",
            Password: "", // no password set
            DB:       0,  // use default DB
        })
        RedisClient := client

        var timeStampList []int64
        for {
            buf := make([]byte, 4096)
            n, err := conn.Read(buf)
            if n == 0 && err != nil { // simplified
                break
            }

            var data Data
            if err = json.Unmarshal(buf[23:n], &data); err == nil {
                height := data.Header.Height
                chainId := data.Header.ChainId
                timestamp := time.Now().Unix()

                key := "kda:" + strconv.FormatInt(height, 10) + ":" + strconv.FormatInt(chainId, 10)
                value := strconv.FormatInt(timestamp, 10)
                fmt.Println("key:", key)
                fmt.Println("value:", value)

                _, err := redisOperation.SetEX(RedisClient, key, "3600", value).Result()
                if err != nil {
                    fmt.Println("redis set ltc data error=", err.Error())
                }
                healthCheck <- 1003
                if len(timeStampList) > 10 {
                    timeStampList = timeStampList[1:]
                    //fmt.Println(healthList)
                }
                timeStampList = append(timeStampList, timestamp)

            } else {
                fmt.Printf("An error occurred in the Node:%s, error is %s \n", "111", err)
                //healthCheck <- 0
            }
        }
    }()
}

type Data struct {
    TxCount int64  `json:"txCount"`
    PowHash string `json:"powHash"`
    Header  struct {
        CreationTime int64  `json:"creationTime"`
        Parent       string `json:"parent"`
        Height       int64  `json:"height"`
        Hash         string `json:"hash"`
        ChainId      int64  `json:"chainId"`
        Weight       string `json:"weight"`
        FeatureFlags int64  `json:"featureFlags"`
        EpochStart   int64  `json:"epochStart"`
        Adjacents    struct {
            Field1 string `json:"5"`
            Field2 string `json:"3"`
            Field3 string `json:"6"`
        } `json:"adjacents"`
        PayloadHash     string `json:"payloadHash"`
        ChainwebVersion string `json:"chainwebVersion"`
        Target          string `json:"target"`
        Nonce           string `json:"nonce"`
    } `json:"header"`
    Target string `json:"target"`
}

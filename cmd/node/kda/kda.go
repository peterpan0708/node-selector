package kda

import (
    "bytes"
    "context"
    "crypto/tls"
    "encoding/json"
    "fmt"
    "github.com/asmcos/requests"
    "gopkg.in/redis.v5"
    "io"
    "log"
    "net"
    "net/http"
    "node-selector/configs"
    redisOperation "node-selector/tools/redis"
    "strconv"
    "time"
)

type Node struct {
    RedisClient *redis.Client
    ctx         context.Context
    ctxCancel   func()
    NetClient   *net.TCPConn
}

type CutResp struct {
    Hashes map[string]struct {
        Height int64 `json:"height"`
    } `json:"hashes"`
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

func (node *Node) Start(config configs.Node) {
    client := redis.NewClient(&redis.Options{
        Addr:     "127.0.0.1:6379",
        Password: "", // no password set
        DB:       0,  // use default DB
    })
    node.RedisClient = client

    lanUrls := config.Urls.Lan
    wanUrls := config.Urls.Wan
    urls := wanUrls
    for _, url := range lanUrls {
        urls = append(urls, url)
    }

    go node.DiagnoseNode("47.101.48.191")

    //// 诊断所有节点的健康性,包括将updates数据写入redis内
    //for _, url := range urls {
    //    fmt.Println(url)
    //    go node.DiagnoseNode(url)
    //}

    //// 统计局域网节点的出块时间,通过updates
    //for _, lanUrl := range lanUrls {
    //    fmt.Println(lanUrl)
    //    go node.getStatistics(lanUrl)
    //}
    //
    //// 统计非局域网节点的出块时间,通过updates
    //for _, wanUrl := range wanUrls {
    //    fmt.Println(wanUrl)
    //    go node.getStatistics(wanUrl)
    //}
}

// GetBestNode 获取最优节点
//  具体流程包括，先获取全网流程是否崩溃,再看主节点是否健康,
func (node *Node) GetBestNode() {

}

// DiagnoseNode 诊断节点的健康性,存入redis中
// 包括 444/1848 端口是否畅通
// 1848高度是否停滞
// 444推送是否堵塞
func (node *Node) DiagnoseNode(url string) {
    var healthList []int64
    healthCheck := make(chan int64)
    go func() {
        count := 0
        for x := range healthCheck {
            fmt.Println(x)
            if x == 1001 {
                //fmt.Println("GetNodeHeight健康")
            }
            if x == 1002 {
                //fmt.Println("GetMiningWork健康")
            }
            if x == 1003 {
                //fmt.Println("GetUpdatesWithFunc健康")
            }
            if x == 0 {
                fmt.Println("不健康")
            }

            if count > 50 {
                healthScore := 100
                for _, v := range healthList {
                    if v == 0 {
                        healthScore--
                    }
                }
                fmt.Println("健康得分:", healthScore)
                count = 0
            }
            if len(healthList) > 100 {
                healthList = healthList[1:]
                //fmt.Println(healthList)
            }
            healthList = append(healthList, x)
            count++
        }
    }()

    // 建立1848 updates 长连接
    node.GetUpdatesWithFunc(url, node.HandleConnection, healthCheck)

    // 1848 一次性get work
    node.GetMiningWork(url, healthCheck)

    //444 获取高度
    node.GetNodeHeight(url, healthCheck)

    time.Sleep(1000 * time.Second)
}

func (node *Node) GetNodeHeight(url string, healthCheck chan int64) {
    duration := time.Second * 1
    timer := time.NewTimer(duration)
    go func() {
        for {
            select {
            case <-timer.C:
                var res CutResp
                req := requests.Requests()
                tr := &http.Transport{
                    TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
                }
                req.Client.Transport = tr
                // 超时时间
                req.SetTimeout(1)

                resp, err := req.Get("https://" + url + ":444/chainweb/0.0/mainnet01/cut")
                if err != nil {
                    fmt.Printf("failed to get %s block height, err: %v \n", url, err.Error())
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
func (node *Node) GetMiningWork(url string, healthCheck chan int64) {
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
                // 超时时间
                http_client := &http.Client{Timeout: time.Second}
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
func (node *Node) GetUpdatesWithFunc(url string, handleConnection func(conn io.ReadCloser, healthCheck chan int64), healthCheck chan int64) {
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
    }
    handleConnection(response.Body, healthCheck)
    return
}

// HandleConnection 直接在这里写个定时器,1s出10个就判定出错
func (node *Node) HandleConnection(conn io.ReadCloser, healthCheck chan int64) {
    go func() {
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

                _, err := redisOperation.SetEX(node.RedisClient, key, "3600", value).Result()
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

//// isPortAccessible 端口是否通畅
//func (n *Node) isPortAccessible(url string) {
//
//}
//
//func (n *Node) isRpcPortAccessible(url string) {
//
//}
//
//func (n *Node) isP2PPortAccessible(url string) {
//
//}

// isHeightBlocked 高度是否停滞
func (node *Node) isHeightBlocked() {

}

// isLongConnectionBlock 推送是否堵塞
func (node *Node) isLongConnectionBlock() {

}

// isCrashed 全网是否崩溃
func (node *Node) isCrashed() {

}

// GetFastestNode 获取最快的节点
func (node *Node) GetFastestNode() {

}

func (node *Node) getStatistics(url string) {

}

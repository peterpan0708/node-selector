package kda

import (
    "bufio"
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
    "strings"
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

//
// Start
//  @Description: 开启统计模块
//  @receiver node
//  @param config
//  @param chanData
//
func (node *Node) Start(config configs.Node, chanData chan interface{}) {
    node.ctx, node.ctxCancel = context.WithCancel(context.Background())

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

    // 诊断所有节点的健康性,包括将updates数据写入redis内
    for _, url := range urls {
        fmt.Println(url)
        key := "kda:health:" + url
        _, err := redisOperation.Set(node.RedisClient, key, "0").Result()
        if err != nil {
            fmt.Println("redis set kda health data error=", err.Error())
            return
        }
        healthScores := make(chan int64)

        go node.DiagnoseNode(url, healthScores, config.Account)
        go func(url string) {
            for healthScore := range healthScores {
                if healthScore < 98 {
                    fmt.Println(url, "不健康")
                    _, err := redisOperation.Set(node.RedisClient, key, "0").Result()
                    if err != nil {
                        fmt.Println("redis set kda health data error=", err.Error())
                        return
                    }
                    // 判断该url是否正在使用
                    node.isActivity(url, urls, lanUrls, wanUrls, chanData)
                } else {
                    fmt.Println(url, "健康")
                    _, err := redisOperation.Set(node.RedisClient, key, "1").Result()
                    if err != nil {
                        fmt.Println("redis set kda health data error=", err.Error())
                        return
                    }
                }
            }
        }(url)
    }

    // 定时任务 计算延迟
    go node.calculateDelay(urls, lanUrls, wanUrls)
}

//
// GetBestNode
//  @Description: 获取最优节点,具体流程包括，先获取全网流程是否崩溃,再看主节点是否健康.
//  @receiver node
//  @return string
//
func (node *Node) GetBestNode() string {
    fastestNode, err := redisOperation.Get(node.RedisClient, "kda:fastestNode").Result()
    if err != nil {
        fmt.Println("redis get err=", err)
        return fastestNode
    } else {
        return ""
    }
}

//
// isActivity
//  @Description: 判断该节点是否可用,如果不可用,则推送一个新的给长连接
//  @receiver node
//  @param url
//  @param urls
//  @param lanUrls
//  @param wanUrls
//  @param chanData
//
func (node *Node) isActivity(url string, urls []string, lanUrls []string, wanUrls []string, chanData chan interface{}) {
    fastestNode := node.GetBestNode()
    if url == fastestNode {
        node.calculateDelay(urls, lanUrls, wanUrls)
        newFastestNode := node.GetBestNode()
        chanData <- newFastestNode
    }
    return
}

// DiagnoseNode
//  @Description: 诊断节点的健康性,存入redis中,包括 444/1848 端口是否畅通,1848高度是否停滞,444推送是否堵塞
//  @receiver node
//  @param url
//  @param healthScore
//
func (node *Node) DiagnoseNode(url string, healthScore chan int64, account configs.Account) {
    var healthList []int64
    healthCheck := make(chan int64)
    go func() {
        count := 0
        for x := range healthCheck {
            //fmt.Println(x)
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
                var healthScoreCal int64
                healthScoreCal = 100
                for _, v := range healthList {
                    if v == 0 {
                        healthScoreCal--
                    }
                }
                fmt.Println("健康得分:", healthScoreCal)
                healthScore <- healthScoreCal
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
    ctx, cancel := context.WithCancel(context.Background())
    node.GetUpdatesWithFunc(url, node.HandleConnection, healthCheck, cancel)

    go func(ctx context.Context) {
        for {
            select {
            case <-ctx.Done():
                println("need reconnect")
                ctx, cancel = context.WithCancel(context.Background())
                node.GetUpdatesWithFunc(url, node.HandleConnection, healthCheck, cancel)
            }
        }
    }(ctx)

    // 1848 一次性get work
    go node.GetMiningWork(url, healthCheck, account)

    //444 获取高度
    go node.GetNodeHeight(url, healthCheck)

    time.Sleep(1000 * time.Second)
}

//
// GetNodeHeight
//  @Description: 通过444端口获取节点高度
//  @receiver node
//  @param url
//  @param healthCheck
//
func (node *Node) GetNodeHeight(url string, healthCheck chan int64) {
    duration := time.Second * 2
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
                    healthCheck <- 0
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

//
// GetMiningWork
//  @Description: 通过1848端口获取mining work
//  @receiver node
//  @param url
//  @param healthCheck
//
func (node *Node) GetMiningWork(url string, healthCheck chan int64, account configs.Account) {
    duration := time.Second * 2
    timer := time.NewTimer(duration)

    go func() {
        for {
            select {
            case <-timer.C:

                body := "{\n    \"account\": \"" + account.Account + "\",\n    \"predicate\": \"keys-all\",\n    \"public-keys\": [\n        \"" + account.PublicKeys + "\"\n    ]\n}"
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
                } else {
                    //defer response.Body.Close()

                    buf := make([]byte, 1024)
                    n, err := response.Body.Read(buf)
                    if n == 0 && err != nil {
                        fmt.Printf("An error occurred in the Node:%s, error is %s \n", url, err)
                        healthCheck <- 0
                        //log.Fatal(err)
                        //return
                    } else {
                        healthCheck <- 1002
                    }
                    //fmt.Println(string(buf[:n]))
                    //fmt.Println(string(buf))
                    //

                    //s := hex.EncodeToString(buf[:n])
                    //fmt.Println(s)
                }
                timer.Reset(duration)
            }
        }
    }()
}

//
// GetUpdatesWithFunc
//  @Description: 建立1848长连接,存入redis内,如果多久没推,就判定有问题
//  @receiver node
//  @param url
//  @param handleConnection
//  @param healthCheck
//  @param cancel
//  @return conn
//
func (node *Node) GetUpdatesWithFunc(url string, handleConnection func(conn io.ReadCloser, healthCheck chan int64, url string, cancel func()), healthCheck chan int64, cancel func()) (conn *http.Response) {
    for {
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
            time.Sleep(5 * time.Second)
        } else {
            go handleConnection(response.Body, healthCheck, url, cancel)
            return response
        }
        //return
    }
}

//
// HandleConnection
//  @Description: 1848长连接的处理,直接在这里写个定时器,1s出10个就判定出错
//  @receiver node
//  @param conn
//  @param healthCheck
//  @param url
//  @param cancel
//
func (node *Node) HandleConnection(conn io.ReadCloser, healthCheck chan int64, url string, cancel func()) {
    //go func() {
    reader := bufio.NewReader(conn)
    var timeStampList []int64
    for {
        if _, isPrefix, err := reader.ReadLine(); err != nil {
            println("failed to read line:", err.Error())
            err := conn.Close()
            if err != nil {
                println(err.Error())
            }
            println("success close tcp connection  ", url)
            //l.ctxCancel()
            cancel()
            return
        } else if isPrefix {
            println("buffer size small")
            return
        } else {
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

                key := "kda:" + url + ":" + strconv.FormatInt(height, 10) + ":" + strconv.FormatInt(chainId, 10)
                value := strconv.FormatInt(timestamp, 10)
                //fmt.Println("key:", key)
                //fmt.Println("value:", value)

                _, err := redisOperation.SetEX(node.RedisClient, key, 3600, value).Result()
                if err != nil {
                    fmt.Println("redis set kda data error=", err.Error())
                }
                healthCheck <- 1003
                if len(timeStampList) > 9 {
                    // 如果1s内出了超过10个块,则判定不健康
                    if timeStampList[len(timeStampList)-1]-timeStampList[0] < 2 {
                        healthCheck <- 0
                        healthCheck <- 0
                        healthCheck <- 0
                    }
                    timeStampList = timeStampList[1:]
                    //fmt.Println(healthList)
                }
                timeStampList = append(timeStampList, timestamp)

            } else {
                fmt.Printf("An error occurred in the Node:%s, error is %s \n", url, err)
                //healthCheck <- 0
            }
        }
    }
    //}()
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

//
// isHealth
//  @Description: 判断当前url是否健康
//  @receiver node
//  @param url
//  @return int64
//
func (node *Node) isHealth(url string) int64 {
    key := "kda:health:" + url
    value, err := redisOperation.Get(node.RedisClient, key).Result()
    if err != nil {
        fmt.Println("redis set kda data error=", err.Error())
        return 0
    }
    health, err := strconv.ParseInt(value, 10, 0)
    return health
}

//// isHeightBlocked 高度是否停滞
//func (node *Node) isHeightBlocked() {
//
//}
//
//// isLongConnectionBlock 推送是否堵塞
//func (node *Node) isLongConnectionBlock() {
//
//}
//
//// isCrashed 全网是否崩溃
//func (node *Node) isCrashed() {
//
//}

//
// GetAndSetFastestNode
//  @Description: 获取当前最快的节点
//  @receiver node
//  @param lanUrls
//  @param wanUrls
//
func (node *Node) GetAndSetFastestNode(lanUrls []string, wanUrls []string) {

    // 各个出块时间相互比较速度,得出最快的节点
    // 计算出health列表
    var healthyLanUrls []string
    for _, lanUrl := range lanUrls {
        if node.isHealth(lanUrl) == 1 {
            healthyLanUrls = append(healthyLanUrls, lanUrl)
        }
    }

    var healthyWanUrls []string
    for _, wanUrl := range wanUrls {
        if node.isHealth(wanUrl) == 1 {
            healthyWanUrls = append(healthyWanUrls, wanUrl)
        }
    }

    if len(healthyLanUrls)+len(healthyWanUrls) <= 0 {
        fmt.Println("全网崩溃")
        _, err := redisOperation.Set(node.RedisClient, "kda:fastestNode", "null").Result()
        if err != nil {
            fmt.Println("redis set err=", err)
            return
        }
    } else if len(healthyLanUrls) == 0 {
        // 在healthyWanUrls中选择最快的
        fmt.Println("局域网内所有节点崩溃")

        delayMin := 100.0
        urlMin := ""
        for _, healthyWanUrl := range healthyWanUrls {
            key := "kda:delay:" + healthyWanUrl
            delayString, err := redisOperation.Get(node.RedisClient, key).Result()
            if err != nil {
                fmt.Println("redis get err=", err)
                break
            }
            delay, err := strconv.ParseFloat(delayString, 64)
            if delay < delayMin {
                delayMin = delay
            }
            urlMin = healthyWanUrl
        }
        if urlMin == "" {
            fmt.Println("GetFastestNode err: cannot get fastest node from wan urls when lan urls are all broken")
        } else {
            _, err := redisOperation.Set(node.RedisClient, "kda:fastestNode", urlMin).Result()
            if err != nil {
                fmt.Println("redis set err=", err)
            }
        }
    } else if len(healthyLanUrls) == 1 {
        _, err := redisOperation.Set(node.RedisClient, "kda:fastestNode", healthyLanUrls[0]).Result()
        if err != nil {
            fmt.Println("redis set err=", err)
        }
    } else if len(healthyLanUrls) > 1 {
        // 在healthyWanUrls中选择最快的

        delayMin := 100.0
        urlMin := ""
        for _, healthyLanUrl := range healthyLanUrls {
            key := "kda:delay:" + healthyLanUrl
            delayString, err := redisOperation.Get(node.RedisClient, key).Result()
            if err != nil {
                fmt.Println("redis get err=", err)
                break
            }
            delay, err := strconv.ParseFloat(delayString, 64)
            if delay < delayMin {
                delayMin = delay
            }
            urlMin = healthyLanUrl
        }
        if urlMin == "" {
            fmt.Println("GetFastestNode err: cannot get fastest node from wan urls when lan urls are all broken")
        } else {
            _, err := redisOperation.Set(node.RedisClient, "kda:fastestNode", urlMin).Result()
            if err != nil {
                fmt.Println("redis set err=", err)
            }
        }
    } else {
        _, err := redisOperation.Set(node.RedisClient, "kda:fastestNode", "null").Result()
        if err != nil {
            fmt.Println("redis set err=", err)
        }
    }

}

//
// calculateDelayCycle
//  @Description: 周期性执行calculateDelay()
//  @receiver node
//  @param urls
//  @param lanUrls
//  @param wanUrls
//
func (node *Node) calculateDelayCycle(urls []string, lanUrls []string, wanUrls []string) {
    duration := time.Second * 300
    timer := time.NewTimer(duration)

    go func() {
        for {
            select {
            case <-timer.C:
                err := node.calculateDelay(urls, lanUrls, wanUrls)
                if err != nil {
                    break
                }
                timer.Reset(duration)
            }
        }
    }()
}

//
// calculateDelay
//  @Description: 通过redis里存储的各个url的出块时间,随便选一个url作为参照物,计算出每个url的delay,存入redis中
//  @receiver node
//  @param urls
//  @param lanUrls
//  @param wanUrls
//
func (node *Node) calculateDelay(urls []string, lanUrls []string, wanUrls []string) error {
    // 读取所有健康的url
    var healthyUrls []string
    for _, url := range urls {
        if node.isHealth(url) == 1 {
            healthyUrls = append(healthyUrls, url)
        }
    }
    if len(healthyUrls) == 0 {
        return nil
    }

    // 随机选一个作为参照物
    referenceUrl := healthyUrls[0]
    // 将其他与参照物作对比
    for _, healthyUrl := range healthyUrls {
        var delays []int64

        // 取出该url的所有
        s := "kda:" + healthyUrl + ":"
        keys, err := redisOperation.Keys(node.RedisClient, s).Result()
        if err != nil {
            fmt.Println("redis keys err=", err)
            return err
        }
        for _, key := range keys {
            timeStampString, err := redisOperation.Get(node.RedisClient, key).Result()
            if err != nil {
                fmt.Println("redis get err=", err)
                return err
            }

            // 寻找参照url的timestamp,如果找不到,就
            referenceKey := strings.Replace(key, healthyUrl, referenceUrl, 1)
            //fmt.Println(referenceKey)
            referenceTimeStampString, err := redisOperation.Get(node.RedisClient, referenceKey).Result()
            if err != nil {
                fmt.Println("redis get err=", err)
                break
            }
            timeStamp, err := strconv.ParseInt(timeStampString, 10, 64)
            referenceTimeStamp, err := strconv.ParseInt(referenceTimeStampString, 10, 64)
            delayOnce := timeStamp - referenceTimeStamp
            delays = append(delays, delayOnce)
        }
        if (len(delays)) > 0 {
            var delaySum int64
            delaySum = 0
            for _, v := range delays {
                delaySum = delaySum + v
            }
            delay := float64(delaySum) / float64(len(delays))

            delayKey := "kda:delay:" + healthyUrl
            delayValue := strconv.FormatFloat(delay, 'f', 4, 64)

            _, err = redisOperation.Set(node.RedisClient, delayKey, delayValue).Result()
            if err != nil {
                fmt.Println("redis set err=", err)
                return err
            }
        }
    }
    fmt.Println("calculateDelay finished")
    node.GetAndSetFastestNode(lanUrls, wanUrls)

    return nil
}

package kda

import (
    "context"
    "encoding/json"
    "fmt"
    "gopkg.in/redis.v5"
    "log"
    "net"
    "net/http"
    "time"
)

type Node struct {
    RedisClint *redis.Client
    ctx context.Context
    ctxCancel func()
    NetClient *net.TCPConn
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

func (n *Node) Start() {

}

// DiagnoseNode 诊断节点的健康性
func (n *Node) DiagnoseNode()  {

}

// isNodePortAccessible 端口是否通畅
func (n *Node) isNodePortAccessible()  {
    
}

// isHeightBlocked 高度是否停滞
func (n *Node) isHeightBlocked()  {
    
}

// isLongConnectionBlock 推送是否堵塞
func (n *Node) isLongConnectionBlock()  {
    
}

// isCrashed 全网是否崩溃
func (n *Node) isCrashed()  {
    
}

// getFastestNode 获取最快的节点
func (n *Node) getFastestNode()  {
    
}




func (n *Node) GetHeight(url string, domain int) (height []int64, chainId []int64, timestamp []int64) {
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
        //log.Fatal(err)
        return
    }

    buf := make([]byte, 4096)
    for {
        n, err := response.Body.Read(buf)
        if n == 0 && err != nil { // simplified
            break
        }

        var data Data
        //fmt.Println("!!!")
        //fmt.Println(string(buf[:n]))
        //fmt.Println("???")
        if err := json.Unmarshal(buf[23:n], &data); err == nil {
           height = append(height, data.Header.Height)
           chainId = append(chainId, data.Header.ChainId)
           timestamp = append(timestamp, time.Now().Unix())
           //fmt.Println("Height:", data.Header.Height)
           //fmt.Println("ChainId:", data.Header.ChainId)
           if len(height) >= domain {
               return
           }
        } else {
           fmt.Printf("An error occurred in the Node:%s, error is %s \n", url, err)
           //fmt.Println(string(buf[:n]))
           //fmt.Println("Unmarshal part:", string(buf[23:n]))
        }
        //fmt.Printf("%s", buf[:n])
    }
    fmt.Println()
    return
}

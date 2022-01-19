package main

import (
    "bytes"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "time"
)

func main() {
    //g, c, t := GetUpdatesWithFunc("47.101.48.191", HandleConnection)
    //fmt.Println(g)
    //fmt.Println(c)
    //fmt.Println(t)

    GetMiningWork("47.101.48.191")
}

func GetMiningWork(url string) {
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
        //log.Fatal(err)
        return
    }
    defer response.Body.Close()

    if err != nil {
        fmt.Printf("An error occurred in the Node:%s, error is %s \n", url, err)
        //log.Fatal(err)
        return
    }

    buf := make([]byte, 1024)
    n, err := response.Body.Read(buf)
    if n == 0 && err != nil {
        fmt.Printf("An error occurred in the Node:%s, error is %s \n", url, err)
        //log.Fatal(err)
        return
    }
    //fmt.Println(string(buf[:n]))
    //fmt.Println(string(buf))
    //
    s := hex.EncodeToString(buf[:n])
    fmt.Println(s)
    return

}

func GetUpdatesWithFunc(url string, handleConnection func(conn io.ReadCloser) (height []int64, chainId []int64, timestamp []int64)) (height []int64, chainId []int64, timestamp []int64) {
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
    height, chainId, timestamp = handleConnection(response.Body)
    return
}

func HandleConnection(conn io.ReadCloser) (height []int64, chainId []int64, timestamp []int64) {
    buf := make([]byte, 4096)
    for {
        n, err := conn.Read(buf)
        if n == 0 && err != nil { // simplified
            break
        }

        var data Data
        //fmt.Println("!!!")
        //fmt.Println(string(buf[:n]))
        //fmt.Println("???")
        if err := json.Unmarshal(buf[23:n], &data); err == nil {
            fmt.Println(data)
            height = append(height, data.Header.Height)
            chainId = append(chainId, data.Header.ChainId)
            timestamp = append(timestamp, time.Now().Unix())
            //fmt.Println("Height:", data.Header.Height)
            //fmt.Println("ChainId:", data.Header.ChainId)
            if len(height) >= 5 {
                return
            }
        } else {
            fmt.Printf("An error occurred in the Node:%s, error is %s \n", "111", err)
            //fmt.Println(string(buf[:n]))
            //fmt.Println("Unmarshal part:", string(buf[23:n]))
        }
        //fmt.Printf("%s", buf[:n])
    }
    fmt.Println()
    return
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

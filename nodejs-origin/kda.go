package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

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

type Kda struct {
}

func main() {
	h1 := make(chan []int64)
	h2 := make(chan []int64)
	h3 := make(chan []int64)

	c1 := make(chan []int64)
	c2 := make(chan []int64)
	c3 := make(chan []int64)

	t1 := make(chan []int64)
	t2 := make(chan []int64)
	t3 := make(chan []int64)

	var k Kda

	go func() {
		url := "http://139.196.143.4:1848"
		h, c, t := k.getHeight(url)
		fmt.Println(url)
		fmt.Println(h, c, t)
		h1 <- h
		c1 <- c
		t1 <- t
	}()
	go func() {
		url := "http://47.101.48.191:1848"
		h, c, t := k.getHeight(url)
		fmt.Println(url)
		fmt.Println(h, c, t)
		h2 <- h
		c2 <- c
		t2 <- t
	}()
	go func() {
		url := "http://54.95.228.181:1848"
		h, c, t := k.getHeight(url)
		fmt.Println(url)
		fmt.Println(h, c, t)
		h3 <- h
		c3 <- c
		t3 <- t
	}()

	hh1 := <-h1
	hh2 := <-h2
	hh3 := <-h3

	cc1 := <-c1
	cc2 := <-c2
	cc3 := <-c3

	tt1 := <-t1
	tt2 := <-t2
	tt3 := <-t3

	fmt.Println("!!!right", hh1)
	fmt.Println("!!!right", hh2)
	fmt.Println("!!!right", hh3)

	fmt.Println("!!!right", cc1)
	fmt.Println("!!!right", cc2)
	fmt.Println("!!!right", cc3)

	fmt.Println("!!!right", tt1)
	fmt.Println("!!!right", tt2)
	fmt.Println("!!!right", tt3)

	score1 := 0
	score2 := 0
	score3 := 0
	var dtime1 int64
	dtime1 = 0
	var dtime2 int64
	dtime2 = 0
	var dtime3 int64
	dtime3 = 0

	for k1, v1 := range hh1 {
		for k2, v2 := range hh2 {
			for k3, v3 := range hh3 {
				if v2 == v1 && cc2[k2] == cc1[k1] && v3 == v1 && cc3[k3] == cc1[k1] {
					if tt1[k1] <= tt2[k2] && tt1[k1] <= tt3[k3] {
						score1 += 1
						fmt.Println(tt2[k2] + tt3[k3] - 2*tt1[k1])
						dtime1 += tt2[k2] + tt3[k3] - 2*tt1[k1]
					}
					if tt2[k2] <= tt1[k1] && tt2[k2] <= tt3[k3] {
						score2 += 1
						fmt.Println(tt1[k1] + tt3[k3] - 2*tt2[k2])
						dtime2 += tt1[k1] + tt3[k3] - 2*tt2[k2]
					}
					if tt3[k3] <= tt2[k2] && tt3[k3] <= tt1[k1] {
						score3 += 1
						fmt.Println(tt2[k2] + tt1[k1] - 2*tt3[k3])
						dtime3 += tt2[k2] + tt1[k1] - 2*tt3[k3]
					}
				}
			}
		}
	}

	fmt.Println("score1:", score1)
	fmt.Println("score2:", score2)
	fmt.Println("score3:", score3)

	fmt.Println("score1_d:", float64(dtime1)/30)
	fmt.Println("score2_d:", float64(dtime2)/30)
	fmt.Println("score3_d:", float64(dtime3)/30)

}

func (kda *Kda) getHeight(url string) (height []int64, chainId []int64, timestamp []int64) {
	request, err := http.NewRequest("GET", url+"/chainweb/0.0/mainnet01/header/updates", nil)
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
		log.Fatal(err)
	}

	buf := make([]byte, 4096)
	for {
		n, err := response.Body.Read(buf)
		if n == 0 && err != nil { // simplified
			break
		}

		var data Data
		if err := json.Unmarshal(buf[23:n], &data); err == nil {
			height = append(height, data.Header.Height)
			chainId = append(chainId, data.Header.ChainId)
			timestamp = append(timestamp, time.Now().Unix())
			//fmt.Println("Height:", data.Header.Height)
			//fmt.Println("ChainId:", data.Header.ChainId)
			if len(height) >= 30 {
				return
			}
		} else {
			fmt.Println(err)
			fmt.Println(url)
			fmt.Println(string(buf[:n]))
		}
		//fmt.Printf("%s", buf[:n])
	}
	fmt.Println()
	return
}

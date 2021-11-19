package kda

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"sync"
	"time"
)

type Selector struct {
}

var k Node

func (s *Selector) Selector(urls []string, domain int) (urlFastest string) {
	duration := time.Second * 60
	timer := time.NewTimer(duration)
	conn, err := redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		fmt.Println("redis.Dial err=", err)
		return
	}
	defer conn.Close()
	go func() {
		for {
			select {
			case <-timer.C:
				var wg sync.WaitGroup

				var heightAll1 [100][]int64
				var chainIdAll1 [100][]int64
				var timestampAll1 [100][]int64
				for key, url := range urls {
					//fmt.Println(key)
					fmt.Println(url)
					wg.Add(1)
					go func(u string, ke int) {
						h, c, t := k.GetHeight(u, domain)
						//fmt.Println(u)
						//fmt.Println(h, c, t)
						heightAll1[ke] = h
						chainIdAll1[ke] = c
						timestampAll1[ke] = t
						wg.Done()
					}(url, key)
				}
				wg.Wait()
				heightAll := heightAll1[0:len(urls)]
				chainIdAll := chainIdAll1[0:len(urls)]
				timestampAll := timestampAll1[0:len(urls)]

				//fmt.Println("!!!!!!!", heightAll)
				//fmt.Println("!!!!!!!", chainIdAll)
				//fmt.Println("!!!!!!!", timestampAll)

				var delay1 [100]float64
				for j := 1; j < len(urls); j++ {
					wg.Add(1)
					go func(i int) {
						//fmt.Println(i)
						var ip1 [][]int64
						var ip2 [][]int64
						ip1 = append(ip1, heightAll[0])
						ip1 = append(ip1, chainIdAll[0])
						ip1 = append(ip1, timestampAll[0])

						ip2 = append(ip2, heightAll[i])
						ip2 = append(ip2, chainIdAll[i])
						ip2 = append(ip2, timestampAll[i])
						delayI := CompareKda(ip1, ip2)
						delay1[i] = delayI
						wg.Done()
					}(j)
				}
				wg.Wait()

				delay := delay1[0:len(urls)]
				//fmt.Println("delay:", delay)
				kmin := 0
				vmin := float64(0)
				for k, v := range delay {
					if v <= vmin {
						kmin = k
						vmin = v
					}
				}

				urlFastest = urls[kmin]
				fmt.Println(urlFastest)
				_, err = conn.Do("Set", "node:kda", urlFastest)
				timer.Reset(duration)
			}
		}
	}()
	return
}

func CompareKda(ip1 [][]int64, ip2 [][]int64) (delay float64) {
	delayAll := int64(0)
	l := len(ip1[0])

	for k1, v1 := range ip1[0] {
		for k2, v2 := range ip2[0] {
			if v1 == v2 && ip1[1][k1] == ip2[1][k2] {
				delayAll += ip2[2][k2] - ip1[2][k1]
			}
		}
	}
	delay = float64(delayAll) / float64(l)
	return
}

package main

import (
	"fmt"
	uuid "github.com/satori/go.uuid"
	"sync"
	"time"
)

type ReqInfo struct {
	KeyIndex int         // 签名索引
	HashData string      // 签名数据
	RetSign  chan string // 签名返回值
}

var (
	MaxCount   = 125                      // 条进行打包
	TimePeriod = 100                      // 多少个时间单位
	ReqMap     = make(map[string]ReqInfo) // 请求信息包
	ReqMu      = new(sync.Mutex)          // 请求数据锁
)

// 克隆数据，并将源数据删除
func cloneAndDelSrcMap(src *map[string]ReqInfo) map[string]ReqInfo {
	cloneData := make(map[string]ReqInfo)
	for k, v := range *src {
		cloneData[k] = v
		delete(*src, k)
	}
	return cloneData
}

// 定时器：到一定时间进行发送处理
func TimeService() {
	for {
		time.Sleep(time.Microsecond * time.Duration(TimePeriod))

		ReqMu.Lock()

		count := len(ReqMap)
		if count > 0 {
			fmt.Printf("定时发送%d\n", count)
			desMap := cloneAndDelSrcMap(&ReqMap)
			go sendReq(desMap)
		}

		ReqMu.Unlock()
	}
}

// 请求入队列，达到一定数量就进行发送
func ReqToQueue(req ReqInfo) {
	ReqMu.Lock()

	key := fmt.Sprintf("%s", uuid.Must(uuid.NewV4(), nil))
	ReqMap[key] = req

	if len(ReqMap) == MaxCount {
		fmt.Println("到MaxCount了，开始发送一批")
		desMap := cloneAndDelSrcMap(&ReqMap)
		go sendReq(desMap)
	}

	ReqMu.Unlock()

}

// 定时发送数据
func sendReq(batReq map[string]ReqInfo) error {
	var err error = nil

	for key, value := range batReq {
		value.RetSign <- key + "=" + value.HashData
	}
	return err
}

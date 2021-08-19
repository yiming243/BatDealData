
package main

import (
	"fmt"
	"strconv"
	"time"
)

func workThread(index int, hashData string) (string, error) {
	req := ReqResInfo{KeyIndex: int32(index), HashData:hashData, RetSign:make(chan string)}

	go ReqToQueue(&req)

	if msg, ok := <-req.RetSign; ok {
		fmt.Printf("index:%v, req.index:%v, sign:%v, \n",  index, req.KeyIndex, msg)
	}

	close(req.RetSign)

	return "", nil
}

func main() {
	// 开启定时处理任务
	go TimeService()
	go TimeDelService()

	for i := 0; i < 10000; i++ {
		// 模拟，正常情况会判断返回值等等
		go workThread(i, strconv.Itoa(i))
	}

	time.Sleep(time.Second * 1111111)
	fmt.Println("over")

}


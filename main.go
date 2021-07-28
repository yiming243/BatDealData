
package main

import (
	"fmt"
	"strconv"
	"time"
)

func workThread(index int, hashData string) (string, error) {
	SignRes := make(chan string)

	req := ReqInfo{KeyIndex: index, HashData: hashData, RetSign: SignRes}
	go ReqToQueue(req)

	if msg, ok := <-SignRes; ok {
		fmt.Printf("msg:%v, index:%v\n", msg, index)
	}

	close(SignRes)

	return "", nil
}

func main() {

	// 开启定时处理任务
	go TimeService()

	for i := 0; i < 100; i++ {
		// 模拟，正常情况会判断返回值等等
		go workThread(i, strconv.Itoa(i))
	}

	time.Sleep(time.Second * 11)
	fmt.Println("over")
}


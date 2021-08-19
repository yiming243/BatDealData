/*
流程有点繁琐，写个说明
功能：
	批量处理数据，数据可以按照一定数量，或者到一定时间后批量发送。
实现原理：
	1. 服务开启式，启动批量发送定时器，启动删除定时器。
	2. 多线程业务时，业务进入到队列中，保存在 map->list 中，当发送一批数据后 map中这一批次
       数据的批量号，放到删除队列中（延迟1秒放入）。
	3. 删除定时器会定时1秒循环，当发现删除队列中有数据，则删除map中的这一项，循环结束后，清空删除队列。
备注：
	业务处理后续可以根据自己的需要自己调整。
 */

package main

import (
	"errors"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"sync"
	"time"
)

var (
	MaxCount   = 100               		    // 条进行打包
	TimeSendPeriod = 10                     // 多少个时间单位(毫秒)
	ReqMu      = new(sync.Mutex)            // 请求数据锁
	GBatReqMap = make(map[string]BatchReq)  // 保存所有请求信息，以及返回信息（返回信息会定期清理）
	GBatReq BatchReq                        // 当前操作的map对象的一个数据
	MapKey     = ""                         // 保存当前处理GBatReq 的key值

	GDelList   = make([]string,0)           // 删除列表
	DelMu      = new(sync.Mutex)            // 删除数据锁
)

type BatchReq struct{
	BatchNumber string		// 批次号
	ReqList []*ReqResInfo 	// 请求
	TotalCount  int16       // ReqList存放数据数量
}

type ReqResInfo struct {
	SeqNum   int16		 // 队列号      2字节
	KeyIndex int32       // 签名索引    4字节
	AuthCode string      // 授权码      16字节
	HashData string      // 摘要值      32字节
	ErrCode  int32       // 错误码      4字节
	RetSign  chan string // 签名返回值   73字节
}  // 共计131字节


// 请求入队列，达到一定数量就进行发送
func ReqToQueue(req *ReqResInfo) {
	ReqMu.Lock() // 加锁
	defer ReqMu.Unlock() // 解锁

	if len(MapKey) == 0 {
		MapKey = fmt.Sprintf("%s", uuid.Must(uuid.NewV4(), nil))
		GBatReq.BatchNumber = MapKey
		GBatReq.TotalCount = 1  // 重新计数
		GBatReq.ReqList = nil   // 清空数组内容
		GBatReq.ReqList = append(GBatReq.ReqList, req)
	}else{
		GBatReq.TotalCount = GBatReq.TotalCount + 1
		GBatReq.ReqList = append(GBatReq.ReqList, req)
	}

	// 更新数据
	req.SeqNum = GBatReq.TotalCount

	if GBatReq.TotalCount >= int16(MaxCount) {
		fmt.Println("到MaxCount了，开始发送一批")
		// 放入map中
		GBatReqMap[MapKey]  = GBatReq
		//重置
		MapKey = ""

		go sendReq(GBatReq)
	}
}

// 定时删除已处理数据
func TimeDelService() {
	for{
		time.Sleep(time.Second * time.Duration(1))

		if len(GDelList) <= 0 {
			continue
		}

		DelMu.Lock()
		ReqMu.Lock()

		for _, v := range GDelList{
			delete(GBatReqMap, v)
		}
		//fmt.Printf("len(GBatReqMap) count:%v, key:%v\n", len(GBatReqMap), "")
		GDelList = GDelList[:0] // 清空

		ReqMu.Unlock()
		DelMu.Unlock()
	}

}

// 请求数组转换为buf
func ReqStructToBuf(req []*ReqResInfo) []byte {
	res := make([]byte, 0)
	authCode := make([]byte, 16)
	hashCode := make([]byte, 32)
	for _, v := range req{
		SeqNumB := Int16ToBytes(v.SeqNum)
		KeyIndexB := Int32ToBytes(v.KeyIndex)
		//BytesCombine(SeqNumB[0:2], KeyIndexB[0:4], []byte(v.AuthCode)[0:16], []byte(v.HashData)[0:32])
		oneByte := BytesCombine(SeqNumB[0:2], KeyIndexB[0:4], authCode[0:16], hashCode[0:32])
		//fmt.Printf("\noneByte:%v", oneByte)
		res = append(res, oneByte...)
	}
	return res
}

// 返回结构转换为结构
func ResBufToStruct(resBuf []byte, res []*ReqResInfo) error {
	//seq_num[2]; err_code[4]; sig_len[2]; sig_value[73];
	sepCount := 54
	count := len(resBuf) / sepCount
	for i:=0; i< count; i++ {
		dst := make([]byte, sepCount)
		copy(dst, resBuf[(i*sepCount):((i+1)*sepCount)])
		seqNum := BytesToInt16(dst[0:2])
		index, err := FindArrIndex(seqNum, res)
		if err != nil{
			return err
		}
		res[index].ErrCode = int32(seqNum)
		//res[index].ErrCode = BytesToInt32(dst[2:6])
		//sigLen := BytesToInt16(dst[6:8])
		//RetSign, err := SignDerData2RS(dst[8:(8+sigLen)])

		RetSign := []byte("1111222")
		res[index].RetSign <- string(RetSign[:])
	}
	return nil
}

// 定时发送数据
func sendReq(batReq BatchReq) error {
	//SendMu.Lock()
	//defer SendMu.Lock()

	reqByte := ReqStructToBuf(batReq.ReqList)
	//fmt.Printf("reqByte:%v\n", reqByte)

	// 解析返回值
	err := ResBufToStruct(reqByte, batReq.ReqList)
	if err != nil{
		return err
	}

	// 处理完毕后，将此批数据的批次号写入到待删除列表中
	go writeDelSignal(batReq.BatchNumber)

	fmt.Printf("\nlen(GBatReqMap) begin:%v\n", len(GBatReqMap))

	//for index, value := range batReq {
	//	batReq[index].RetSign <- strconv.Itoa(int(value.SeqNum)) + "=" + value.HashData
	//}


	return err
}

// 将此批数据的批次号写入到待删除列表中
func writeDelSignal(key string){
	time.Sleep(time.Second * time.Duration(1))

	DelMu.Lock()
	defer DelMu.Unlock()
	GDelList = append(GDelList, key)
}


func FindArrIndex(seqNum int16, res []*ReqResInfo) (int, error){
	for i,v := range res{
		if v.SeqNum == seqNum {
			return i, nil
		}
	}
	return 0, errors.New("not exist")
}



// 定时器：到一定时间进行发送处理
func TimeService() {
	for {
		time.Sleep(time.Microsecond * time.Duration(TimeSendPeriod))

		ReqMu.Lock()

		if GBatReq.TotalCount > 0 && len(MapKey) > 0 {
			fmt.Printf("定时发送%d\n", GBatReq.TotalCount)

			// 放入map中
			GBatReqMap[MapKey]  = GBatReq
			//重置
			MapKey = ""
			go sendReq(GBatReq)
		}

		ReqMu.Unlock()
	}
}




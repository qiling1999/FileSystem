package main

import (
	"bufio"
	"encoding/json"
	"FileSystem/config"
	dblayer "FileSystem/db"
	"FileSystem/mq"
	"FileSystem/store/oss"
	"log"
	"os"
)

// ProcessTransfer : 处理文件转移
func ProcessTransfer(msg []byte) bool {
	//1.解析msg
	log.Println(string(msg))

	pubData := mq.TransferData{}
	err := json.Unmarshal(msg, &pubData)
	if err != nil {
		log.Println(err.Error())
		return false
	}

	//2.根据msg里面的信息找到文件当前临时存储的路径，创建文件句柄。
	fin, err := os.Open(pubData.CurLocation)
	if err != nil {
		log.Println(err.Error())
		return false
	}

	//3.通过文件句柄，将文件读取出来并写入到oss上面去。
	err = oss.Bucket().PutObject(
		pubData.DestLocation,
		bufio.NewReader(fin))
	if err != nil {
		log.Println(err.Error())
		return false
	}

	//更新数据库中文件表的存储路径
	_ = dblayer.UpdateFileLocation(
		pubData.FileHash,
		pubData.DestLocation)
	return true
}

func main() {
	if !config.AsyncTransferEnable {
		log.Println("异步转移文件功能目前被禁用，请检查相关配置")
		return
	}
	log.Println("文件转移服务启动中，开始监听转移任务队列...")
	mq.StartConsume(
		config.TransOSSQueueName,
		"transfer_oss",
		ProcessTransfer)
}

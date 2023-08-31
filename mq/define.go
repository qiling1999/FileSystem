package mq

import (
	cmn "FileSystem/common"
)

// TransferData : 将要写到rabbitmq的数据的结构体
//转移队列中消息载体的结构格式
type TransferData struct {
	FileHash      string
	CurLocation   string   //临时存储的地址
	DestLocation  string   //要转移的目标的地址
	DestStoreType cmn.StoreType  //表明要转移到的存储类型，local？cepha？oss?
}

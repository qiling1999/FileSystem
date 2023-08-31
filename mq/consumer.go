package mq

import "log"

var done chan bool

// StartConsume : 接收消息
func StartConsume(qName, cName string, callback func(msg []byte) bool) {
	//1.通过channel.Consume获得消息信道，，这个信道是golang里面的一个channel，没有缓冲区的channel在存消息和取消息的时候都是阻塞的。
	msgs, err := channel.Consume(
		qName,
		cName,
		true,  //自动应答
		false, // 非唯一的消费者
		false, // rabbitMQ只能设置为false
		false, // noWait, false表示会阻塞直到有消息过来
		nil)
	if err != nil {
		log.Fatal(err)
		return
	}

	done = make(chan bool)

	//2.循环从channel里获取队列消息
	go func() {
		// 循环读取channel的数据
		for d := range msgs {
			//3.调用callback方法来处理新的消息
			processErr := callback(d.Body)
			if processErr {
				// TODO: 将任务写入错误队列，待后续处理
			}
		}
	}()

	// 接收done的信号, 没有信息过来则会一直阻塞，避免该函数退出
	<-done

	// 关闭通道
	channel.Close()
}

// StopConsume : 停止监听队列
func StopConsume() {
	done <- true
}

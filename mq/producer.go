package mq

import (
	"FileSystem/config"
	"log"

	"github.com/streadway/amqp"
)

var conn *amqp.Connection  //RabbitMQ的连接对象
var channel *amqp.Channel  //通过channel来进行消息的发布和接收

// 如果异常关闭，会接收通知
var notifyClose chan *amqp.Error

func init() {
	// 是否开启异步转移功能，开启时才初始化rabbitMQ连接
	if !config.AsyncTransferEnable {
		return
	}
	if initChannel() {
		channel.NotifyClose(notifyClose)
	}
	// 断线自动重连
	go func() {
		for {
			select {
			case msg := <-notifyClose:
				conn = nil
				channel = nil
				log.Printf("onNotifyChannelClosed: %+v\n", msg)
				initChannel()
			}
		}
	}()
}

//初始化channel
func initChannel() bool {
	//1.判断channel是否已经创建过
	if channel != nil {
		return true
	}

	//2.获得rabbitmq的一个连接
	conn, err := amqp.Dial(config.RabbitURL)
	if err != nil {
		log.Println(err.Error())
		return false
	}

	//3.打开一个channel，用于消息的发布与接收等
	channel, err = conn.Channel()
	if err != nil {
		log.Println(err.Error())
		return false
	}
	return true
}

// Publish : 发布消息
func Publish(exchange, routingKey string, msg []byte) bool {
	//1.判断channel是否是正常的
	if !initChannel() {
		return false
	}
	//2.通过channel执行消息发布动作
	if nil == channel.Publish(
		exchange,
		routingKey,
		false, // 如果没有对应的queue, 就会丢弃这条消息
		false, //
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        msg}) {
		return true
	}
	return false
}

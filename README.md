# FileSystem
基于Golang实现的分布式文件云存储系统，类似百度网盘，可以上传文件到本地存储或者Ceph私有云或者阿里OSS公有云

功能简介：
可以选择将文件上传到本地存储或者Ceph私有云或者阿里OSS公有云，或者从中下载文件
文件系统：文件分块上传，断点续传，秒传，文件下载，文件修改，文件删除，
用户系统：用户注册/登录，查询用户信息，支持用户session鉴权，拦截器验证token，用户/数据的资源隔离

技术亮点：
使用docker部署MySQL主从节点架构，来存储用户文件表和唯一文件表。
使用redis数据库来存储文件分块信息，实现分块上传即断点续传。
使用docker部署Ceph分布式存储系统搭建私有云，支持将文件上传到Ceph私有云。
连接阿里云OSS，实现文件从私有云迁移到公有云，保证用户量很大的情况下，高并发带来的可用性问题。
使用RabbitMQ消息队列，实现将服务模块中的同步逻辑转化成异步来执行，解决逻辑耦合，处理高并发和大规模消息问题。
使用gin框架加入中间件，并改造由原生net/http实现的路由规则和接口功能，以此来实现应用架构微服务化。

MySQL相关表创建：
从db/table.sql中查找。

需要手动安装的库：
go get github.com/garyburd/redigo/redis
go get github.com/go-sql-driver/mysql
#go get github.com/garyburd/redigo/redis
go get github.com/gomodule/redigo/redis
go get github.com/json-iterator/go
go get github.com/aliyun/aliyun-oss-go-sdk/oss
go get gopkg.in/amz.v1/aws
go get gopkg.in/amz.v1/s3
go get github.com/streadway/amqp
go get github.com/gin-gonic/gin
go get github.com/gin-contrib/cors

在加入rabbitMQ实现文件异步转移阶段，启动方式(分裂成了两个独立程序)：
启动上传应用程序:
> cd $GOPATH/<你的工程目录>
> cd $GOPATH/filestore-server
> go run service/upload/main.go
启动转移应用程序:
> cd $GOPATH/<你的工程目录>
> cd $GOPATH/filestore-server
> go run service/transfer/main.go

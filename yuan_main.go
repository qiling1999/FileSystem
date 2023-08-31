package main

import (
	"fmt"
	"net/http"
	"FileSystem/handler"
)

//最初的主函数，在加入消息队列，异步转移队列功能之后，
//分裂成了两个独立的应用，上传服务和转移服务
//main函数转移到了service/upload/main.go
//在系统微服务化和使用gin框架之后，upload/main.go中的路由规则被独立出来
//放到了route/router.go，存放路由规则表，upload/main.go只需要调用router.go里面的方法就可以了

func main() {
	http.HandleFunc("/file/upload", handler.UploadHandler)//用这个方法来建立一个路由规则
	http.HandleFunc("/file/upload/suc",handler.UploadSucHandler)//上传成功的路由
	http.HandleFunc("/file/meta",handler.GetFileMetaHandler)//获取文件元信息的路由，当成查询单个文件的路由
	http.HandleFunc("/file/query", handler.FileQueryHandler)//批量查询文件元信息的路由
	http.HandleFunc("/file/download", handler.DownloadHandler)//下载文件的路由
	http.HandleFunc("/file/update",handler.FileMetaUpdateHandler)//更新文件的路由(重命名)
	http.HandleFunc("/file/delete",handler.FileDeleteHandler)//删除文件的路由

	// 秒传接口
	http.HandleFunc("/file/fastupload", handler.HTTPInterceptor(
		handler.TryFastUploadHandler))

	//生成文件下载地址URL，然后点击下载地址跳转到下载页面
	http.HandleFunc("/file/downloadurl", handler.HTTPInterceptor(
		handler.DownloadURLHandler))


	// 分块上传接口
	http.HandleFunc("/file/mpupload/init",
		handler.HTTPInterceptor(handler.InitialMultipartUploadHandler))
	http.HandleFunc("/file/mpupload/uppart",
		handler.HTTPInterceptor(handler.UploadPartHandler))
	http.HandleFunc("/file/mpupload/complete",
		handler.HTTPInterceptor(handler.CompleteUploadHandler))

	// 用户相关接口
	http.HandleFunc("/user/signup",handler.SignupHandler)//用户注册的路由
	http.HandleFunc("/user/signin",handler.SignInHandler)//用户登录的路由
	http.HandleFunc("/user/info",handler.UserInfoHandler)//用户信息查询的路由
	http.HandleFunc("/user/info",handler.HTTPInterceptor(handler.UserInfoHandler))//增加拦截器的用户信息查询的路由

	//
	//路由规则完成之后，就开始端口监听工作
	err := http.ListenAndServe(":8080",nil)
	if err != nil {
		fmt.Printf("Flie to start server,err:%s", err.Error())
		return
	}
}




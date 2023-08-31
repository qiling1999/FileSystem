package route

import (
	"FileSystem/handler"

	"github.com/gin-gonic/gin"
)

// Router : 路由表配置
func Router() *gin.Engine {
	// gin framework, 包括Logger, Recovery
	router := gin.Default()

	// 处理静态资源
	router.Static("/static/", "./static")

	// 不需要经过验证就能访问的接口
	router.GET("/user/signup", handler.SignupHandler)//响应用户注册页面路由
	router.POST("/user/signup", handler.DoSignupHandler)//处理用户注册post请求

	router.GET("/user/signin", handler.SignInHandler)//响应用户登录页面路由
	router.POST("/user/signin", handler.DoSignInHandler)//处理用户登录post请求

	// 加入中间件，用于校验token的拦截器
	router.Use(handler.HTTPInterceptor())   //http请求拦截器接口

	// Use之后的所有handler都会经过拦截器进行token校验

	// 文件存取接口
	router.GET("/file/upload", handler.UploadHandler)//响应上传页面的接口
	router.POST("/file/upload", handler.DoUploadHandler)//上传文件的接口，处理文件上传
	router.GET("/file/upload/suc", handler.UploadSucHandler)//上传文件成功提示页面处理器
	router.POST("/file/meta", handler.GetFileMetaHandler)//获取文件元信息接口
	router.POST("/file/query", handler.FileQueryHandler)//查询批量的文件元信息接口
	router.GET("/file/download", handler.DownloadHandler)//文件下载接口
	router.POST("/file/update", handler.FileMetaUpdateHandler)//更新元信息接口(重命名)
	router.POST("/file/delete", handler.FileDeleteHandler)//删除文件及元信息
	router.POST("/file/downloadurl", handler.DownloadURLHandler)//生成文件的下载地址接口

	// 秒传接口
	router.POST("/file/fastupload", handler.TryFastUploadHandler)//尝试秒传接口

	// 分块上传接口
	router.POST("/file/mpupload/init", handler.InitialMultipartUploadHandler)//初始化分块上传接口
	router.POST("/file/mpupload/uppart", handler.UploadPartHandler)//上传文件分块接口
	router.POST("/file/mpupload/complete", handler.CompleteUploadHandler)//通知上传合并分块接口

	// 用户相关接口
	router.POST("/user/info", handler.UserInfoHandler)

	return router
}

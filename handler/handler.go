package handler

import (
	cmn "FileSystem/common"
	cfg "FileSystem/config"
	dblayer "FileSystem/db"
	"FileSystem/meta"
	"FileSystem/mq"
	"FileSystem/store/ceph"
	"FileSystem/store/oss"
	"FileSystem/util"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func init() {
	// 目录已存在
	if _, err := os.Stat(cfg.TempLocalRootDir); err == nil {
		return
	}

	// 尝试创建目录
	err := os.MkdirAll(cfg.TempLocalRootDir, 0744)
	if err != nil {
		log.Println("无法创建临时存储目录，程序将退出")
		os.Exit(1)
	}
}

// UploadHandler : 响应上传页面
func UploadHandler(c *gin.Context) {
	data, err := ioutil.ReadFile("./static/view/upload.html")
	if err != nil {
		c.String(404, `网页不存在`)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}

// DoUploadHandler ： 处理文件上传
func DoUploadHandler(c *gin.Context) {
	errCode := 0
	defer func() {
		if errCode < 0 {
			c.JSON(http.StatusOK, gin.H{
				"code": errCode,
				"msg":  "Upload failed",
			})
		}
	}()

	// 1. 从form表单中获得文件内容句柄
	file, head, err := c.Request.FormFile("file")
	if err != nil {
		fmt.Printf("Failed to get form data, err:%s\n", err.Error())
		errCode = -1
		return
	}
	defer file.Close()

	// 2. 把文件内容转为[]byte
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		fmt.Printf("Failed to get file data, err:%s\n", err.Error())
		errCode = -2
		return
	}

	// 3. 构建文件元信息
	fileMeta := meta.FileMeta{
		FileName: head.Filename,
		FileSha1: util.Sha1(buf.Bytes()), //　计算文件sha1
		FileSize: int64(len(buf.Bytes())),
		UploadAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	// 4. 将文件写入临时存储位置
	fileMeta.Location = cfg.TempLocalRootDir + fileMeta.FileSha1 // 临时存储地址
	newFile, err := os.Create(fileMeta.Location)
	if err != nil {
		fmt.Printf("Failed to create file, err:%s\n", err.Error())
		errCode = -3
		return
	}
	defer newFile.Close()

	nByte, err := newFile.Write(buf.Bytes())
	if int64(nByte) != fileMeta.FileSize || err != nil {
		fmt.Printf("Failed to save data into file, writtenSize:%d, err:%s\n", nByte, err.Error())
		errCode = -4
		return
	}

	// 5. 同步或异步将文件转移到Ceph/OSS
	newFile.Seek(0, 0) // 游标重新回到文件头部
	if cfg.CurrentStoreType == cmn.StoreCeph {
		// 文件写入Ceph存储
		data, _ := ioutil.ReadAll(newFile)
		cephPath := "/ceph/" + fileMeta.FileSha1
		_ = ceph.PutObject("userfile", cephPath, data)
		fileMeta.Location = cephPath
	} else if cfg.CurrentStoreType == cmn.StoreOSS {
		// 文件写入OSS存储
		ossPath := "oss/" + fileMeta.FileSha1
		// 判断写入OSS为同步还是异步
		if !cfg.AsyncTransferEnable {
			// TODO: 设置oss中的文件名，方便指定文件名下载
			err = oss.Bucket().PutObject(ossPath, newFile)
			if err != nil {
				fmt.Println(err.Error())
				errCode = -5
				return
			}
			fileMeta.Location = ossPath
		} else {
			// 写入异步转移任务队列
			data := mq.TransferData{
				FileHash:      fileMeta.FileSha1,
				CurLocation:   fileMeta.Location,
				DestLocation:  ossPath,
				DestStoreType: cmn.StoreOSS,
			}
			pubData, _ := json.Marshal(data)
			pubSuc := mq.Publish(
				cfg.TransExchangeName,
				cfg.TransOSSRoutingKey,
				pubData,
			)
			if !pubSuc {
				// TODO: 当前发送转移信息失败，稍后重试
			}
		}
	}

	//6.  更新文件表记录
	_ = meta.UpdateFileMetaDB(fileMeta)

	// 7. 更新用户文件表
	username := c.Request.FormValue("username")
	suc := dblayer.OnUserFileUploadFinished(username, fileMeta.FileSha1,
		fileMeta.FileName, fileMeta.FileSize)
	if suc {
		c.Redirect(http.StatusFound, "/static/view/home.html")
	} else {
		errCode = -6
	}
}

// UploadSucHandler : 上传已完成
func UploadSucHandler(c *gin.Context) {
	c.JSON(http.StatusOK,
		gin.H{
			"code": 0,
			"msg":  "Upload Finish!",
		})
}

/*
//定义一个用于上传文件的接口，
//然后第一个参数是用于向用户返回数据的response writer对象
//第二个参数是用于接收用户请求的一个request对象指针
func UploadHandler(w http.ResponseWriter, r *http.Request){
	//然后首先来判断用户请求的HTTP方法是什么是GET还是POST
	if r.Method == "GET"{
		//返回上传html页面，static静态目录/view
		//然后就是写逻辑，向用户返回我们的HTML页面的内容
		data, err := ioutil.ReadFile("src/project/FileSystem/static/view/index.html")//ioutil.ReadFile()将文件给加载出来，返回文件内容和err
		if err != nil {
			io.WriteString(w, "internel server error")
			return
		}
		io.WriteString(w, string(data))//如果成功的话，将文件的内容给返回去
		//上传接口的GET规则已经完成了，然后在main的方法里面把路由规则给设定好
	}else if r.Method == "POST"{
		//接收用户上传的文件流及存储到本地目录
		//实现对post请求方法的处理，因为客服端现在用的是表单的形式来提交
		file, head, err := r.FormFile("file")//通过这个方法取到表单里面的文件
		if err != nil {
			fmt.Printf("Failed to get data, err:%s\n", err.Error())
			return
		}
		defer file.Close() //在函数退出之前要把文件的句柄给关掉

		//在创建本地文件之前，先新建一个filemeta对象
		fileMeta := meta.FileMeta{
			FileName: head.Filename,
			Location: "/tmp/" + head.Filename,
			UploadAt: time.Now().Format("2006-01-02 15:04:05"),
		}

		//创建本地的文件来接受文件流
		newFile, err := os.Create(fileMeta.Location)//创建文件句柄
		if err != nil {
			fmt.Printf("Failed to create file, err:%s\n", err.Error())
			return
		}
		defer newFile.Close()//在函数退出之前要把文件的句柄给关掉

		//第3步操作就是将内存中的文件流的内容拷贝到新的文件的缓冲区中去
		fileMeta.FileSize, err = io.Copy(newFile, file)
		if err != nil {
			fmt.Printf(

				"Failed to save data into file, err:%v\n", err.Error())
			return
		}
		//然后计算上传文件的哈希值也就是Sha1值
		//我们需要将file句柄的seek位置移到最前面，也就是0的位置
		newFile.Seek(0, 0)
		//然后通过调用util.FileSha1的方法，将句柄传进去，然后就会得到文件的一个Sha1值
		fileMeta.FileSha1 = util.FileSha1(newFile)

		// 5. 同步或异步将文件转移到Ceph/OSS
		newFile.Seek(0, 0) // 游标重新回到文件头部
		if cfg.CurrentStoreType == cmn.StoreCeph {
			// 文件写入Ceph存储
			data, _ := ioutil.ReadAll(newFile)   //将文件全部读取出来
			cephPath := "/ceph/" + fileMeta.FileSha1  //这样可以保证用户上传的对象存在ceph里面对应的key的一个唯一性
			_ = ceph.PutObject("userfile", cephPath, data)
			fileMeta.Location = cephPath
		} else if cfg.CurrentStoreType == cmn.StoreOSS {
			// 文件写入OSS存储
			ossPath := "oss/" + fileMeta.FileSha1//这样可以保证用户上传的对象存在oss里面对应的key的一个唯一性
			//同步的逻辑
			//err = oss.Bucket().PutObject(ossPath, newFile)
			//if err != nil {
			//	fmt.Println(err.Error())
			//	w.Write([]byte("Upload failed!"))
			//	return
			//}
			//fileMeta.Location = ossPath
			//
			//异步转移的逻辑
			// 判断写入OSS为同步还是异步
			if !cfg.AsyncTransferEnable {
				//同步转移逻辑
				err = oss.Bucket().PutObject(ossPath, newFile)
				if err != nil {
					fmt.Println(err.Error())
					w.Write([]byte("Upload failed!"))
					return
				}
				fileMeta.Location = ossPath
			} else {
				// 写入异步转移任务队列
				data := mq.TransferData{
					FileHash:      fileMeta.FileSha1,
					CurLocation:   fileMeta.Location,
					DestLocation:  ossPath,
					DestStoreType: cmn.StoreOSS,
				}
				pubData, _ := json.Marshal(data)
				pubSuc := mq.Publish(
					cfg.TransExchangeName,
					cfg.TransOSSRoutingKey,
					pubData,
				)
				if !pubSuc {
					// TODO: 当前发送转移信息失败，稍后重试
				}
			}
		}

		fmt.Println(fileMeta.Location)

		//TODO：更新用户文件表记录
		//将fileMeta信息添加到meta集合中去
		meta.UpdateFileMeta(fileMeta)
		//将fileMeta信息添加到MySQL中去
		_ = meta.UpdateFileMetaDB(fileMeta)

		//上面已经将接收到的文件内容已经完成拷贝，并且写到本地的文件目录里面去了
		//下面可以向客户端返回一些正确的信息，或者可以做一个重定向到一个提示成功的页面
		//在handler里面做一次重定向，然后在main里面将这个路由给加上去
		r.ParseForm()
		username := r.Form.Get("username")
		//username需要通过解析请求的参数来获取
		suc := dblayer.OnUserFileUploadFinished(username, fileMeta.FileSha1,
			fileMeta.FileName, fileMeta.FileSize)
		if suc {
			http.Redirect(w, r, "/static/view/home.html", http.StatusFound)
		} else {
			w.Write([]byte("Upload Failed."))
		}
	}
}


//上传文件成功提示页面处理器
func UploadSucHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "文件上传成功！")
}
*/

// GetFileMetaHandler : 获取文件元信息
func GetFileMetaHandler(c *gin.Context) {
	filehash := c.Request.FormValue("filehash")
	fMeta, err := meta.GetFileMetaDB(filehash)
	if err != nil {
		c.JSON(http.StatusInternalServerError,
			gin.H{
				"code": -2,
				"msg":  "Upload failed!",
			})
		return
	}

	if fMeta != nil {
		data, err := json.Marshal(fMeta)
		if err != nil {
			c.JSON(http.StatusInternalServerError,
				gin.H{
					"code": -3,
					"msg":  "Upload failed!",
				})
			return
		}
		c.Data(http.StatusOK, "application/json", data)
	} else {
		c.JSON(http.StatusOK,
			gin.H{
				"code": -4,
				"msg":  "No such file",
			})
	}
}

/*
//GetFileMetaHandler 获取文件元信息
func GetFileMetaHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()//需要解析客户端请求的参数r.ParseForm()方法进行解析操
	filehash := r.Form["filehash"][0]//获取参数，假设客户端上传的用来查询文件元信息的参数叫做filehash
	//fMeta := meta.GetFileMeta(filehash)//然后从存好的文件元信息集合里面获取对应数据
	fMeta, err := meta.GetFileMetaDB(filehash)//从mysql中里面获取对应数据
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	data, err := json.Marshal(fMeta)	//因为这时候返回的是fMeta的结构对象，需要转换成json string格式返回给客户端

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(data)
}*/

// FileQueryHandler : 查询批量的文件元信息
func FileQueryHandler(c *gin.Context) {
	limitCnt, _ := strconv.Atoi(c.Request.FormValue("limit"))
	username := c.Request.FormValue("username")
	userFiles, err := dblayer.QueryUserFileMetas(username, limitCnt)
	if err != nil {
		c.JSON(http.StatusInternalServerError,
			gin.H{
				"code": -1,
				"msg":  "Query failed!",
			})
		return
	}

	data, err := json.Marshal(userFiles)
	if err != nil {
		c.JSON(http.StatusInternalServerError,
			gin.H{
				"code": -2,
				"msg":  "Query failed!",
			})
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

/*
// FileQueryHandler : 查询批量的文件元信息
func FileQueryHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	limitCnt, _ := strconv.Atoi(r.Form.Get("limit"))
	username := r.Form.Get("username")
	userFiles, err := dblayer.QueryUserFileMetas(username, limitCnt)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(userFiles)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
 		return
	}
	w.Write(data)
}
*/

// DownloadHandler : 文件下载接口
func DownloadHandler(c *gin.Context) {
	fsha1 := c.Request.FormValue("filehash")
	username := c.Request.FormValue("username")
	// TODO: 处理异常情况
	fm, _ := meta.GetFileMetaDB(fsha1)
	userFile, _ := dblayer.QueryUserFileMeta(username, fsha1)

	if strings.HasPrefix(fm.Location, cfg.TempLocalRootDir) {
		// 本地文件， 直接下载
		c.FileAttachment(fm.Location, userFile.FileName)
	} else if strings.HasPrefix(fm.Location, "/ceph") {
		// ceph中的文件，通过ceph api先下载
		bucket := ceph.GetCephBucket("userfile")
		data, _ := bucket.Get(fm.Location)
		//	c.Header("content-type", "application/octect-stream")
		c.Header("content-disposition", "attachment; filename=\""+userFile.FileName+"\"")
		c.Data(http.StatusOK, "application/octect-stream", data)
	}
}

/*
// DownloadHandler : 文件下载
func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()//需要解析客户端请求的参数r.ParseForm()方法进行解析操
	fsha1 := r.Form.Get("filehash")//获取参数，假设客户端上传的用来查询文件元信息的参数叫做filehash
	username := r.Form.Get("username")
	//得到Sha1值之后，到meta集合里面取到对应的文件的元信息
	fm, _ := meta.GetFileMetaDB(fsha1)//从唯一文件表中获取对应的文件元信息
	userFile, err := dblayer.QueryUserFileMeta(username, fsha1)//从用户文件表中查找对应的信息
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var fileData []byte
	if strings.HasPrefix(fm.Location, cfg.TempLocalRootDir) {
		f, err := os.Open(fm.Location)  //打开文件存储地址
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer f.Close()

		//得到文件句柄之后，就需要将这个文件的内容给读取出来，因为测试的文件都是小文件，所有用ioutil.ReadALL方法，将文件的内容全部加载到内存中去
		//如果文件很大的话，就需要实现一个流的方式，就是每次读一小部分的数据返回给客户端，然后在刷新缓存，继续读到文件的末尾为止。
		fileData, err = ioutil.ReadAll(f)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else if strings.HasPrefix(fm.Location, "/ceph") {
		fmt.Println("to download file from ceph...")
		bucket := ceph.GetCephBucket("userfile")
		fileData, err = bucket.Get(fm.Location)
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	//理论上只需要下面的Write就可以了，为了在浏览器里面做一些演示，需要把http的响应头给扎一下，让浏览器给识别出来，然后就可以当成一个文件的下载
	w.Header().Set("Content-Type", "application/octect-stream")
	// attachment表示文件将会提示下载到本地，而不是直接在浏览器中打开
	w.Header().Set("content-disposition", "attachment; filename=\""+userFile.FileName+"\"")
	//如果都成功的话，就可以用Write方法将这个byte数据给返回去，返回到客户端data
  	w.Write(fileData)
}*/

// FileMetaUpdateHandler ： 更新元信息接口(重命名)
func FileMetaUpdateHandler(c *gin.Context) {
	opType := c.Request.FormValue("op")
	fileSha1 := c.Request.FormValue("filehash")
	username := c.Request.FormValue("username")
	newFileName := c.Request.FormValue("filename")

	if opType != "0" || len(newFileName) < 1 {
		c.Status(http.StatusForbidden)
		return
	}

	// 更新用户文件表tbl_user_file中的文件名，tbl_file的文件名不用修改
	_ = dblayer.RenameFileName(username, fileSha1, newFileName)

	// 返回最新的文件信息
	userFile, err := dblayer.QueryUserFileMeta(username, fileSha1)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	data, err := json.Marshal(userFile)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, data)
}

/*
// FileMetaUpdateHandler ： 更新元信息接口(重命名)
func FileMetaUpdateHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()//解析客户端请求的参数列表，
	//客户端会带三个操作请求，
	//一个是待操作的类型，opType
	//第二个是文件的唯一标志，哈希值 filehash
	//第三个是文件更新后的文件名  filename

	opType := r.Form.Get("op")
	fileSha1 := r.Form.Get("filehash")
	username := r.Form.Get("username")
	newFileName := r.Form.Get("filename")

	if opType != "0" || len(newFileName) < 1 {
		w.WriteHeader(http.StatusForbidden)  //403的错误码
		return
	}
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)  //405的错误码
		return
	}

	// 更新用户文件表tbl_user_file中的文件名，tbl_file的文件名不用修改
	_ = dblayer.RenameFileName(username, fileSha1, newFileName)

	// 返回最新的文件信息
	userFile, err := dblayer.QueryUserFileMeta(username, fileSha1)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	//
	////获取到三个参数之后，需要获取当前文件的元信息
	//curFileMeta := meta.GetFileMeta(fileSha1)//用fileSha1查询
	//curFileMeta.FileName = newFileName
	//meta.UpdateFileMeta(curFileMeta)//然后用更新方法，更新文件元信息
	//w.WriteHeader(http.StatusOK)
	//data, err := json.Marshal(curFileMeta)//转换成json格式的数据返回给客户端
	data, err := json.Marshal(userFile)//转换成json格式的数据返回给客户端
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}*/

// FileDeleteHandler : 删除文件及元信息
func FileDeleteHandler(c *gin.Context) {
	username := c.Request.FormValue("username")
	fileSha1 := c.Request.FormValue("filehash")

	fm, err := meta.GetFileMetaDB(fileSha1)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	// 删除本地文件
	os.Remove(fm.Location)
	// TODO: 可考虑删除Ceph/OSS上的文件
	// 可以不立即删除，加个超时机制，
	// 比如该文件10天后也没有用户再次上传，那么就可以真正的删除了

	// 删除文件表中的一条记录
	suc := dblayer.DeleteUserFile(username, fileSha1)
	if !suc {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusOK)
}

/*
// FileDeleteHandler : 删除文件元信息
func FileDeleteHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.Form.Get("username")
	filesha1 := r.Form.Get("filehhash")
	//fMeta := meta.GetFileMeta(filesha1)
	//获取到filesha1之后需要将
	//os.Remove(fMeta.Location)//要在下面这个操作之前取出文件地址，然后删除原文件
	// 删除本地文件
	fm, err := meta.GetFileMetaDB(filesha1)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	os.Remove(fm.Location)
	// 删除文件表中的一条记录
	suc := dblayer.DeleteUserFile(username, filesha1)
	if !suc {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//meta.RemoveFileMeta(filesha1)//删除元信息，但这只是删除一个索引的数据，真正的文件还没有删除
	w.WriteHeader(http.StatusOK)
}*/

// TryFastUploadHandler : 尝试秒传接口
func TryFastUploadHandler(c *gin.Context) {
	// 1. 解析请求参数
	username := c.Request.FormValue("username")
	filehash := c.Request.FormValue("filehash")
	filename := c.Request.FormValue("filename")
	filesize, _ := strconv.Atoi(c.Request.FormValue("filesize"))

	// 2. 从文件表中查询相同hash的文件记录
	fileMeta, err := meta.GetFileMetaDB(filehash)
	if err != nil {
		fmt.Println(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	// 3. 查不到记录则返回秒传失败
	if fileMeta == nil {
		resp := util.RespMsg{
			Code: -1,
			Msg:  "秒传失败，请访问普通上传接口",
		}
		c.Data(http.StatusOK, "application/json", resp.JSONBytes())
		return
	}

	// 4. 上传过则将文件信息写入用户文件表， 返回成功
	suc := dblayer.OnUserFileUploadFinished(
		username, filehash, filename, int64(filesize))
	if suc {
		resp := util.RespMsg{
			Code: 0,
			Msg:  "秒传成功",
		}
		c.Data(http.StatusOK, "application/json", resp.JSONBytes())
		return
	}
	resp := util.RespMsg{
		Code: -2,
		Msg:  "秒传失败，请稍后重试",
	}
	c.Data(http.StatusOK, "application/json", resp.JSONBytes())
	return
}

/*
// TryFastUploadHandler : 尝试秒传接口
func TryFastUploadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	// 1. 解析请求参数
	username := r.Form.Get("username")
	filehash := r.Form.Get("filehash")
	filename := r.Form.Get("filename")
	filesize, _ := strconv.Atoi(r.Form.Get("filesize"))

	// 2. 从文件表中查询相同hash的文件记录
	fileMeta, err := meta.GetFileMetaDB(filehash)
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// 3. 查不到记录则返回秒传失败
	if fileMeta.FileSha1 == "" {
		resp := util.RespMsg{
			Code: -1,
			Msg:  "秒传失败，请访问普通上传接口",
		}
		w.Write(resp.JSONBytes())
		return
	}

	// 4. 上传过则将文件信息写入用户文件表， 返回成功
	suc := dblayer.OnUserFileUploadFinished(
		username, filehash, filename, int64(filesize))
	if suc {
		resp := util.RespMsg{
			Code: 0,
			Msg:  "秒传成功",
		}
		w.Write(resp.JSONBytes())
		return
	}
	resp := util.RespMsg{
		Code: -2,
		Msg:  "秒传失败，请稍后重试",
	}
	w.Write(resp.JSONBytes())
	return
}*/

// DownloadURLHandler : 生成文件的下载地址
func DownloadURLHandler(c *gin.Context) {
	filehash := c.Request.FormValue("filehash")
	// 从文件表查找记录
	row, _ := dblayer.GetFileMeta(filehash)

	// TODO: 判断文件存在OSS，还是Ceph，还是在本地
	if strings.HasPrefix(row.FileAddr.String, cfg.TempLocalRootDir) ||
		strings.HasPrefix(row.FileAddr.String, "/ceph") {
		username := c.Request.FormValue("username")
		token := c.Request.FormValue("token")
		tmpURL := fmt.Sprintf("http://%s/file/download?filehash=%s&username=%s&token=%s",
			c.Request.Host, filehash, username, token)
		c.Data(http.StatusOK, "octet-stream", []byte(tmpURL))
	} else if strings.HasPrefix(row.FileAddr.String, "oss/") {
		// oss下载url
		signedURL := oss.DownloadURL(row.FileAddr.String)
		fmt.Println(row.FileAddr.String)
		c.Data(http.StatusOK, "octet-stream", []byte(signedURL))
	}
}

/*
// DownloadURLHandler : 生成文件的下载地址
func DownloadURLHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	filehash := r.Form.Get("filehash")
	// 从文件表查找记录
	row, _ := dblayer.GetFileMeta(filehash)  //得到文件的哈希值

	// TODO: 判断文件存在OSS，还是Ceph，还是在本地
	if strings.HasPrefix(row.FileAddr.String, "/tmp") ||
		strings.HasPrefix(row.FileAddr.String, "/ceph") {
		username := r.Form.Get("username")
		token := r.Form.Get("token")
		tmpURL := fmt.Sprintf("http://%s/file/download?filehash=%s&username=%s&token=%s",
			r.Host, filehash, username, token)
		w.Write([]byte(tmpURL))
	} else if strings.HasPrefix(row.FileAddr.String, "oss/") {
		// oss下载url
		signedURL := oss.DownloadURL(row.FileAddr.String)
		w.Write([]byte(signedURL))
	}
}*/
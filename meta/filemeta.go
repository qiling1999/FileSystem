package meta

import (
	mydb "FileSystem/db"
)

//FileMeta:文件元信息结构
type FileMeta struct {
	FileSha1 string    //作为文件的唯一标志，也可以用MD5这些等等
	FileName string
	FileSize int64
	Location string    //文件路径
	UploadAt string    //时间戳
}

//定义一个对象存储所有上传文件的元信息，然后每一个上传的元信息就是上面那个结构体
var fileMetas map[string]FileMeta   //string就是这个唯一标识，就是这个Sha1

//fileMetas初始化的工作
func init() {
	fileMetas = make(map[string]FileMeta)
}

//提供一个接口UpdateFileMeta:新增/更新文件元信息
func UpdateFileMeta(fmeta FileMeta) {
	fileMetas[fmeta.FileSha1] = fmeta  //做一次简单的赋值操作
}

//接口UpdateFileMetaDB:新增/更新文件元信息到mysql中
func UpdateFileMetaDB(fmeta FileMeta) bool {
	return mydb.OnFileUploadFinished(fmeta.FileSha1, fmeta.FileName, fmeta.FileSize, fmeta.Location)
}

//接口GetFileMetaDB:从mysql获取文件元信息
func GetFileMetaDB(fileSha1 string) (*FileMeta, error) {
	tfile, err := mydb.GetFileMeta(fileSha1)
	if tfile == nil || err != nil {
		return nil, err
	}
	fmeta := FileMeta{
		FileSha1: tfile.FileHash,
		FileName: tfile.FileName.String,
		FileSize: tfile.FileSize.Int64,
		Location: tfile.FileAddr.String,
	}
	return &fmeta, nil
}

//GetFileMeta:通过Sha1获取文件的元信息对象
func GetFileMeta(fileSha1 string) FileMeta {
	return fileMetas[fileSha1]
}

// RemoveFileMeta : 删除元信息
func RemoveFileMeta(fileSha1 string) {
	delete(fileMetas, fileSha1)
}

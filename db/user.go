package db

import (
	mydb "FileSystem/db/mysql"
	"fmt"
)

// User : 用户表model  与数据库中用户表一一对应
type User struct {
	Username     string
	Email        string
	Phone        string
	SignupAt     string
	LastActiveAt string
	Status       int
}

// UserSignup : 通过用户名及密码完成user表的注册操作
func UserSignup(username string, passwd string) bool {
	stmt, err := mydb.DBConn().Prepare(  //prepare方法写sql语句
		"insert ignore into tbl_user (`user_name`,`user_pwd`) values (?,?)")
	if err != nil {
		fmt.Println("Failed to insert, err:" + err.Error())
		return false
	}
	defer stmt.Close()

	//sql语句的执行
	ret, err := stmt.Exec(username, passwd)  //sql语句的执行，把两个参数给传进去
	if err != nil {
		fmt.Println("Failed to insert, err:" + err.Error())
		return false
	}
	//校验是否已经插入成功了，即校验是否重复注册，重复注册也算失败
	if rowsAffected, err := ret.RowsAffected(); nil == err && rowsAffected > 0 {
		return true
	}//校验是否重复注册
	return false
}

// UserSignin : 判断密码是否一致
func UserSignin(username string, encpwd string) bool {
	//DBConn().Prepare生成一个声明
	stmt, err := mydb.DBConn().Prepare("select * from tbl_user where user_name=? limit 1")
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	defer stmt.Close()

	//执行查询操作， rows代表返回的记录
	rows, err := stmt.Query(username)
	if err != nil {
		fmt.Println(err.Error())
		return false
	} else if rows == nil {
		fmt.Println("username not found: " + username)
		return false
	}

	pRows := mydb.ParseRows(rows)
	if len(pRows) > 0 && string(pRows[0]["user_pwd"].([]byte)) == encpwd {
		return true
	}
	return false
}

// UpdateToken : 刷新用户登录的token  ，将生成的token写到数据库里面去
func UpdateToken(username string, token string) bool {
	//用DBCoon().Prepare方法生成一个声明，即写数据库操作字段
	stmt, err := mydb.DBConn().Prepare(
		"replace into tbl_user_token (`user_name`,`user_token`) values (?,?)")
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	defer stmt.Close()

	//下面就是执行这个Updata操作了
	_, err = stmt.Exec(username, token)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	return true
}

// GetUserInfo : 查询用户信息    返回的是一条对应用户信息的记录
func GetUserInfo(username string) (User, error) {
	user := User{}

	stmt, err := mydb.DBConn().Prepare(
		"select user_name,signup_at from tbl_user where user_name=? limit 1")
	if err != nil {
		fmt.Println(err.Error())
		return user, err
	}
	defer stmt.Close()

	// 执行查询的操作
	err = stmt.QueryRow(username).Scan(&user.Username, &user.SignupAt)
	if err != nil {
		return user, err
	}
	return user, nil
}

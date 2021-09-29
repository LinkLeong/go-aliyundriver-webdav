package main

import (
	"context"
	"flag"
	"fmt"
	"go-aliyun-webdav/aliyun"
	"go-aliyun-webdav/aliyun/cache"
	"go-aliyun-webdav/aliyun/model"
	"go-aliyun-webdav/webdav"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

func init() {
	cache.Init()
}

var Version = "v1.0.3"

type Task struct {
	Id string `json:"id"`
}

func GetDb() *gorm.DB {
	// 参考 https://github.com/go-sql-driver/mysql#dsn-data-source-name 获取详情
	//dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?charset=utf8mb4&parseTime=True&loc=Local", m.User, m.PWD, m.IP, m.Port, m.DBName)
	//db, err := gorm.Open(mysql2.Open(dsn), &gorm.Config{})
	db, err := gorm.Open(sqlite.Open("./db/casaOS.db"), &gorm.Config{})
	c, _ := db.DB()
	c.SetMaxIdleConns(10)
	c.SetMaxOpenConns(100)
	c.SetConnMaxIdleTime(time.Second * 1000)
	if err != nil {
		fmt.Println("连接数据失败!")
		panic("数据库连接失败")
		return nil
	}
	err = db.AutoMigrate(&Task{})
	if err != nil {
		fmt.Println("检查和创建数据库出错", err)
	}
	return db
}

func main() {
	GetDb()
	var port *string
	var path *string
	var refreshToken *string
	//var user *string
	//var pwd *string
	//
	port = flag.String("addr", "8086", "默认8085")
	path = flag.String("path", "./", "")
	//	user = flag.String("user", "admin", "用户名")
	//	pwd = flag.String("pwd", "123456", "密码")
	//refreshToken = flag.String("rt", "a4d7e58c0f7949cb9c88670d9fb00a30", "refresh_token")
	refreshToken = flag.String("rt", "", "refresh_token")
	flag.Parse()

	if len(*refreshToken) == 0 || len(os.Args) < 3 || os.Args[1] != "-rt" {
		fmt.Println("rf为必填项,请输入refreshToken")
		return
	}
	if len(os.Args) > 2 && os.Args[1] == "rt" {
		*refreshToken = os.Args[2]
	}
	var address string
	if runtime.GOOS == "windows" {
		address = ":" + *port
	} else {
		address = "0.0.0.0:" + *port
	}

	//todo 判断

	refreshResult := aliyun.RefreshToken(*refreshToken)

	config := model.Config{
		RefreshToken: refreshResult.RefreshToken,
		Token:        refreshResult.AccessToken,
		DriveId:      refreshResult.DefaultDriveId,
		ExpireTime:   time.Now().Unix() + refreshResult.ExpiresIn,
	}

	fs := &webdav.Handler{
		Prefix:     "/",
		FileSystem: webdav.Dir(*path),
		LockSystem: webdav.NewMemLS(),
		Config:     config,
	}

	//fmt.p

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		// 获取用户名/密码
		//username, password, ok := req.BasicAuth()
		//if !ok {
		//	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		//	w.WriteHeader(http.StatusUnauthorized)
		//	return
		//}
		////	 验证用户名/密码
		//if username != *user || password != *pwd {
		//	http.Error(w, "WebDAV: need authorized!", http.StatusUnauthorized)
		//	return
		//}

		// Add CORS headers before any operation so even on a 401 unauthorized status, CORS will work.

		w.Header().Set("Access-Control-Allow-Origin", "*")

		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE,UPDATE")

		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if req.Method == "GET" && strings.HasPrefix(req.URL.Path, fs.Prefix) {
			info, err := fs.FileSystem.Stat(context.TODO(), strings.TrimPrefix(req.URL.Path, fs.Prefix))
			if err == nil && info.IsDir() {
				req.Method = "PROPFIND"

				if req.Header.Get("Depth") == "" {
					req.Header.Add("Depth", "1")
				}
			}
		}

		fs.ServeHTTP(w, req)
	})
	http.ListenAndServe(address, nil)
}

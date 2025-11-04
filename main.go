package main

import (
	"fmt"

	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"

	"github.com/tommywijayac/duck-queue-server-v2/backend/controllers"
	"github.com/tommywijayac/duck-queue-server-v2/backend/databases"
	"github.com/tommywijayac/duck-queue-server-v2/backend/routers"
)

func main() {
	if err := databases.InitCache(); err != nil {
		logs.Error("Failed to initialize Cache: %s", err.Error())
		panic(err)
	}
	logs.Info("Cache initialized successfully")

	if err := databases.InitRedisClient(); err != nil {
		logs.Error("Failed to initialize Redis client: %v", err)
		panic(err)
	}
	logs.Info("Redis client initialized successfully")

	controllers.Init()
	logs.Info("Controllers initialized successfully")

	routers.Init()
	logs.Info("Routers initialized successfully")

	// somehow using host in WSL2 cause server to be unreachable from Win
	//
	// host, err := web.AppConfig.String("http_host")
	// if err != nil {
	// 	host = "127.0.0.1"
	// }
	port, err := web.AppConfig.String("http_port")
	if err != nil {
		port = "3000"
	}

	web.Run(fmt.Sprintf(":%s", port))
}

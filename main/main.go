package main

import (
	"github.com/bdjimmy/pudding"
	"fmt"
	"time"
)

func main(){
	engine := pudding.Default()

	// 添加中间件
	engine.UseFunc(func(c *pudding.Context) {
		start := time.Now()
		c.Set("log_id", 123)
		c.Next()
		fmt.Printf("cost: %v\n", time.Since(start))
	})

	// 注册公共中间件
	engine.Inject("", func(c *pudding.Context) {
		fmt.Println("engine.Inject")
	})

	// 启动group, 内网
	group := engine.Group("/internal", func(c *pudding.Context) {
		fmt.Println("internal")
	})
	group.GET("/test", func(c *pudding.Context) {
		logId, _ := c.Get("log_id")
		c.String(200, "hello world!, logid=%v", logId)
	})

	// 启动group，外网
	outGroup := engine.Group("/external")
	outGroup.GET("/test", func(c *pudding.Context) {
		logId, _ := c.Get("log_id")
		c.JSON(0, "success", fmt.Sprintf("hello world!, logid=%v", logId))
	})

	// 启动Server
	err := engine.Run(":9090")
	fmt.Println(err)
}

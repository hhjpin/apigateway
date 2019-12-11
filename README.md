### Api Gateway

####环境准备

1. docker > 17.10 & docker-compose > 3.5
2. go 1.10

####项目编译方式

1. 切换到项目目录下
2. bash make.sh $version
3. version tag 根据 registry.henghajiang.com 私有仓库中当前镜像版本决定

#### 代码结构说明

```
|
|--- conf							// 配置文件工具方法包
|--- core							// gateway核心组件
|--- --- constant					// 核心组件常量定义
|--- --- --- bytes.go				// 常量byte slice版本(性能优化)
|--- --- --- const.go				// 常量定义
|--- --- options					// gateway核心组件配置选项
|--- --- --- option.go				// gateway核心组件配置选项结构体定义
|--- --- routing					// 路由及其相关组件
|--- --- --- base.go				
|--- --- --- etcd.go				// etcd存取方法
|--- --- --- health_check.go		// 健康检查组件
|--- --- --- proxy.go				// 代理模块, 处理预置/后置中间件
|--- --- --- routing.go				// 路由模块
|--- --- --- tables.go				// 协程安全的各式路由表定义
|--- --- utils
|--- --- --- utils.go				// 工具方法集合
|--- --- watcher					// Watcher组件
|--- --- --- base.go				
|--- --- --- endpoint.go			// endpoint/node watcher 定义
|--- --- --- health_check.go		// healthCheck watcher 定义
|--- --- --- route.go				// route watcher 定义
|--- --- --- service.go				// service watcher 定义
|--- --- --- watcher.go				// Watcher Interface
|--- dashboard						// 监控平台Api
|--- middleware						// 中间件组件
|--- --- auth.go					// 认证中间件
|--- --- count.go					// 计数器中间件
|--- --- limit.go					// 限流中间件
|--- --- base.go					// 中间件 Middleware Interface
|--- sdk							
|--- --- golang					
|--- --- --- const.go
|--- --- --- gw_sdk.go
|--- --- python
|--- --- --- const.py
|--- --- --- gw_sdk.py
|--- server.go						// Server 入口
```

#### 优化说明

1. server.go: line 121, 使用了 Tcp SO_REUSEPORT Option, 若后续部署时系统不支持该 Tcp 选项, 可以更换为普通Listener
2. 不要在proxy.go中打印任何关于经由Api Gateway的请求报文的string类型日志, 大批量的打印日志将会导致频繁触发GC, 降低处理能力

#### 备注

1. 请求的超时时间由两部分组成: 一是所有中间件中的最大处理时间; 二是转发请求的处理时间
2. api gateway关闭了对 connection:keep-alive的支持, 若需要该特性, 可以考虑其他替代方案(comet/websocket/HTTP2)
3. 新的预置中间件只需满足middleware/base.go中的 Middleware Interface, 即可在gateway初始化时绑定至RequestWrapperHandler; 所有中间件是并发无序执行, 不应期待不同中间件的执行顺序(中间件被缓存在一个队列中, 虽然执行开始的时间近乎相同, 但结束时间不一定相同); 你可以在Work函数的第一个参数 ctx *fasthttp.RequestCtx中拿到所有这次请求相关的数据, 甚至可以通过 ctx.UserValue("Table") 拿到全局的路由表

#### Sdk使用方式

```go
	node := golang.NewNode("127.0.0.1", *flagPort, golang.NewHealthCheck(
		"/check",
		10,
		5,
		3,
		true,
	))
	svr := golang.NewService("test", node)
	gw := golang.NewApiGatewayRegistrant(
		ConnectToEtcd(),
		node,
		svr,
		[]*golang.Router{
			golang.NewRouter("test1", "/front/$1", "/test/$1", svr),
			golang.NewRouter("test2", "/api/v1/test/$1", "/test/$1", svr),
			golang.NewRouter("test3", "/rd", "/redirect", svr),
		},
	)
	go func(g *golang.ApiGatewayRegistrant) {
		if err := g.Register(); err != nil {
			logger.Error(err)
		}
	}(&gw)
```

在 http server启动之前, 完成对 node, service, gateway对象的初始化<br/>

其中Router的语法为:

1. 参数1, router name, 同名的router会被视为同一个Api
2. 参数2, frontend api, 对外Api 路由, '$x'代表正则替换的第x个URL Pattern
3. 参数3, backend api, 内部Api 路由,  '$x'代表正则替换的第x个URL Pattern
4. 参数4, 绑定至的服务对象

```go
func exit(gw *golang.ApiGatewayRegistrant) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	logger.Infof("service exiting ...")
	if err := gw.Unregister(); err != nil {
		logger.Error(err)
	}
	os.Exit(0)
}
```

最后可以通过 exit 函数来实现优雅退出, 这样在服务重启/更新时, 将不会有新的请求抵达该服务
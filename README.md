# ws-simpleIM

基于 websocket 的单机 im 框架

实现群组管理，消息解析，状态/缓存模块，消息推送模块，生命周期钩子，提供简洁 api 快速启动单机 IM 服务，可用于即时通讯数据同步等服务

示例：

最基本的用法，

1. 默认方式实例化 `ws.Engine` 管理类
2. 向 `ws.Engine` 注册示例服务
3. 通过 `gin` 的方式启动 `ws`,实例化 `gin.Engine`，将 `ws.Engine` 的预设方法`ConnectGIN`注册到 `gin.Engine` 中，通过 uri 中的 group 占位符 `:group` 来标识此链接想要加入的群组
4. 启动 gin 监听端口

```go
func TestMain(t *testing.T) {
	r := ws.Default()
	r.Use("post", "/talk", talk)

	g := gin.Default()
	g.GET("/ws/:group", r.ConnectGIN)
	g.Run(":8080")
}

func talk(ctx *ws.Context) {
	msg := &struct {
		Username string `json:"username"`
		Msg      string `json:"msg"`
	}{}
	ctx.BindJSON(msg)
	ctx.BrodcastMe(ws.H{"msg": msg.Username + " say: " + msg.Msg})
}
```

可以看到`Context`提供的两个 api：BindJSON 和 Brodcast 都很类似 gin 所提供的，使用方法确实是类似的。`Context`其余提供的方法一览：

```go
仿gin框架的context上下文结构，以websocket连接所发送一条请求信息为单位 记录此请求所属的连接，连接所属的群组，连接的id，此请求的数据体，响应体（不一定需要响应）

func (c *ws.Context) AddParam(key string, value string)
func (c *ws.Context) BindJSON(obj any) error // json格式解析websocket数据体
func (c *ws.Context) Brodcast(data any) // 发出群内广播
func (c *ws.Context) BrodcastMe(data any) // 发出群内广播（包括自己）
func (c *ws.Context) Cache() treeCache.CacheI
func (c *ws.Context) ConnCache() treeCache.CacheI
func (c *ws.Context) GroupName() string
func (c *ws.Context) IdString() string
func (c *ws.Context) Query(key string) string
func (c *ws.Context) Response(data any) // 单独回复，不广播
func (c *ws.Context) RootCache() treeCache.CacheI
```

生命周期钩子示例：

注册 ConnOpen, ConnClose 等钩子函数

```go
func TestMain(t *testing.T) {
	r := ws.Default()
	r.SetConnOpen(open)
	r.SetConnClose(close)
	g := gin.Default()
	g.GET("/ws/:group", r.ConnectGIN)
	g.Run(":8080")
}

func open(ctx *ws.Context) {
	ctx.BrodcastMe(ws.H{"info": ctx.GroupName() + "-welcome new member!"})
}

func close(ctx *ws.Context) {
	ctx.BrodcastMe(ws.H{"info": ctx.GroupName() + "-a member leave"})
}
```

利用生命周期钩子可以简化很多业务上的代码

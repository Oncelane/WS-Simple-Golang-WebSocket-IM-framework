package ws

import (
	cache "aiosystem-backend/pkg/ws/treeCache"
	"log"
	"net/http"
	"runtime"
	"sync"
	"unsafe"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// 为了实现close回调只调用一次
type ConnStruct struct {
	conn      *websocket.Conn
	onceClose sync.Once
	// onceConnect sync.Once 同一个conn的连接不可能并发，不需要once保证
}

type GroupStruct struct {
	conns map[WSId]*ConnStruct
}

const (
	RecvChanLength = 500
	SendChanLength = 500
)

type Engine struct {
	// next conn id
	globWsId WSId

	// groups
	groups map[string]*GroupStruct

	// msg chan
	revMsgCh  chan *Context
	sendMsgCh chan *Context

	// sync.Map
	cache cache.CacheI

	//servlet
	servlet *Servlet

	// callback
	onConnOpen   HandlerFunc
	onConnClose  HandlerFunc
	onGroupOpen  HandlerFunc
	onGroupClose HandlerFunc

	runOnce sync.Once
	mu      sync.Mutex

	brokerNum int       // broker num
	brokers   []*broker // id to broker
}

// Engin default config
// sync.Map as CachI implement
// callbacks all nil
func Default() (ret *Engine) {
	ret = &Engine{
		globWsId:     0,
		groups:       make(map[string]*GroupStruct),
		revMsgCh:     make(chan *Context, RecvChanLength),
		sendMsgCh:    make(chan *Context, SendChanLength),
		servlet:      NewServelet(),
		cache:        cache.NewSyncCache(),
		onConnOpen:   nil,
		onConnClose:  nil,
		onGroupOpen:  nil,
		onGroupClose: nil,
		brokerNum:    runtime.NumCPU(),
	}
	ret.initBroker()
	return
}

// 将函数注册到对应的方法和路径
func (g *Engine) Use(method string, path string, f HandlerFunc) {
	g.servlet.use(method, path, f)
}

// 通过方法和路径拿取注册的函数
func (g *Engine) getFunc(method string, path string) (HandlerFunc, bool) {
	return g.servlet.getFunc(method, path)
}

// 通过Gname和Wsid获取websocket Connection
// func (g *Engine) getWsConn(gname string, wsid WSId) (*websocket.Conn, bool) {
// 	connst, exist := g.getWsConnSt(gname, wsid)
// 	return connst.conn, exist
// }

// 通过Gname和Wsid获取websocket connect struct
func (g *Engine) getWsConnSt(gname string, wsid WSId) (*ConnStruct, bool) {
	groups, exist := g.groups[gname]
	if !exist {
		return nil, false
	}
	cst, exist := groups.conns[wsid]
	return cst, exist
}

// 开启消息接收和消息发送的后台协程
func (g *Engine) run() {
	// 接收信息,执行在方法+路径上注册的回调函数
	g.Use("control", "unconnect", g.unconnect)

	// 处理请求消息
	go func() {
		ch := g.getRecvMsgCh()
		for ctx := range ch {
			// 解析请求的 method 和 path
			err := ctx.parse()
			if err != nil {
				log.Printf("wsid[%d] method and path decode error %s\n", ctx.wsid, err.Error())
				ctx.Response(H{"msg": "method and path decode error"})
				g.delete(ctx)
				continue
			}

			// 匹配回调函数
			f, ok := g.getFunc(ctx.req.Method, ctx.req.Path)
			if !ok {

				log.Printf("wsid[%d] call method[%s] path:[%s]: callsfunc dose not exist\n", ctx.wsid, ctx.req.Method, ctx.req.Path)
				continue
			}
			log.Printf("wsid[%d] call method[%s] path[%s]\n", ctx.wsid, ctx.req.Method, ctx.req.Path)

			// 异步执行回调函数
			go f(ctx)
		}
	}()

	// 启动broker
	g.runBroker()
}

// 协议升级
var upGrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (g *Engine) addGroup(c *Context) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.groups[c.gname] == nil {
		g.groups[c.gname] = &GroupStruct{
			conns: make(map[WSId]*ConnStruct),
		}

	}
	g.groups[c.gname].conns[c.wsid] = &ConnStruct{
		conn: c.conn,
	}
}

// sync.once保证对同一个connect只调用一次onConnClose回调
// 避免在g.onConnClose中重复触发delete函数造成无限循环调用的问题
func (g *Engine) delete(ctx *Context) {
	log.Println("[Delete]", ctx.wsid)
	cst, exist := g.getWsConnSt(ctx.gname, ctx.wsid)
	if !exist {
		return
	}
	cst.onceClose.Do(func() {
		// 触发回调函数，由于once的存在，不会调用第二次onConnClose回调函数
		if g.onConnClose != nil {

			g.onConnClose(ctx)
		}
		ctx.closeMe(H{"msg": "see you"})

	})
}

// 使用gin方式建立websocket连接
func (g *Engine) ConnectGIN(c *gin.Context) {
	g.runOnce.Do(func() { g.run() })
	// 建立websocket链接
	ws, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"msg": "upgrade failed:" + err.Error()})
		return
	}

	// 通过解析网址，加入选定群组
	gname := c.Param("group")
	if gname == "" {
		// 从get参数中获取group
		gname = c.Query("group")
		if gname == "" {
			c.JSON(http.StatusOK, gin.H{"msg": "missing group name"})
			return
		}
	}

	// 进入监听
	g.handleConnect(ws, gname)

}

// 使用net/http库的方式建立websocket连接
func (g *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("pass2")
	g.runOnce.Do(func() { g.run() })
	// 建立websocket链接
	log.Println("hello?")
	ws, err := upGrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upGrader fail", err)
		return
	}
	// 通过解析网址，加入选定群组
	gname := r.URL.Query().Get("group")
	if gname == "" {
		w.Write([]byte("mising group name"))
		return
	}
	// 进入监听
	log.Println("ws: ", ws, "game", gname)
	g.handleConnect(ws, gname)
}

// 阻塞函数，占用本协程持续进行websocket监听
func (g *Engine) handleConnect(ws *websocket.Conn, gname string) {
	// 分发uuid
	wsid := g.generateUUID()
	ctx := &Context{
		wsid:    wsid,
		message: nil,
		gname:   gname,
		conn:    ws,
		mgr:     g,
	}
	g.addGroup(ctx)
	log.Printf("new connection group=[%s] wsid=[%d]\n", ctx.gname, ctx.wsid)
	// 触发onConnOpen回调，可用于建立pingpang,统计websocket连接信息等等
	if g.onConnOpen != nil {
		g.onConnOpen(ctx)
	}

	// 持续监听socket传入信息
	for {
		// 读取socket中的数据
		_, message, err := ws.ReadMessage()
		if err != nil {
			// 失败
			ctx.message = []byte(err.Error())
			g.delete(ctx)
			break
		}
		// 复制出新的上下文，赋值消息体
		newCtx := &Context{
			wsid:    wsid,
			message: message,
			gname:   gname,
			conn:    ws,
			mgr:     g,
		}

		// 上下文交给分发器，等待调度
		g.getRecvMsgCh() <- newCtx
	}
	log.Printf("group [%s] wsid[%d] exic\n", ctx.gname, ctx.wsid)
}

// 定义网址 method: control , path: unconnect 路径为关闭websocket请求的默认网址
// 目的是用于用于主动触发连接关闭的回调事件
func (g *Engine) unconnect(ctx *Context) {
	log.Printf("group[%s] wsid[%d] call to close\n", ctx.gname, ctx.wsid)
	// 执行删除
	g.delete(ctx)
}

// 设置连接刚建立时的回调函数
func (g *Engine) SetConnOpen(f HandlerFunc) {
	g.onConnOpen = f
}

// 设置连接准备关闭时的回调函数
func (g *Engine) SetConnClose(f HandlerFunc) {
	g.onConnClose = f
}

// 设置群组刚建立时的回调函数
func (g *Engine) SetGroupClose(f HandlerFunc) {
	g.onGroupClose = f
}

// 设置群组准备销毁时的回调函数
func (g *Engine) SetGroupOpen(f HandlerFunc) {
	g.onGroupOpen = f
}

func (g *Engine) getRecvMsgCh() chan *Context {
	return g.revMsgCh
}

func (g *Engine) getSendMsgCh(ptr *websocket.Conn) chan *Context {
	// log.Printf("ptr:%p mod:%d\n", ptr, hashModPtr(unsafe.Pointer(ptr), g.brokerNum))
	return g.brokers[hashModPtr(unsafe.Pointer(ptr), g.brokerNum)].ch
}

func (g *Engine) generateUUID() WSId {
	g.mu.Lock()
	wsid := g.globWsId
	g.globWsId++
	g.mu.Unlock()
	return wsid
}

package ws

import (
	cache "aiosystem-backend/pkg/ws/treeCache"
	"encoding/json"
	"log"
	"strconv"

	"github.com/gorilla/websocket"
)

// 消息发送模式
const (
	BRODCAST = iota
	SINGLE
	BRODCASTME
	CLOSE
)

// 消息类型
const (
	INFO  = "info"
	CACHE = "cache"
	DISK  = "disk"
)

// json format
type H map[string]any

// TODO: 改成专用的uuid分发
type WSId int64

// 仿gin框架的context上下文结构，以websocket连接所发送一条请求信息为单位
// 记录此请求所属的连接，连接所属的群组，连接的id，此请求的数据体，响应体（不一定需要响应）
type Context struct {
	conn        *websocket.Conn // websocket连接
	mgr         *Engine         // 仿gin的engin，管理多个群组
	wsid        WSId            // uuid
	gname       string          // 群组
	message     []byte          // 原始请求消息体
	req         *RequestStruct  // 请求参数+数据
	respMessage []byte          // 序列化响应数据
	resp        *responseStruct // 响应参数+数据
	msgType     int             // 响应类型（广播/一对一/both）
}

func (c *Context) preprocress() {
	newCtx := &Context{
		wsid:    c.wsid,
		gname:   c.gname,
		conn:    c.conn,
		mgr:     c.mgr,
		msgType: c.msgType,
	}
	// log.Println("new Context", newCtx.msgType)
	msg, err := json.Marshal(c.resp)
	if err != nil {
		log.Println("序列化错误")
		return
	}
	newCtx.respMessage = msg

	ch := c.mgr.getSendMsgCh(c.conn)
	ch <- newCtx
	// log.Printf("add ctx type[%d] to ch: ch size[%d]\n", newCtx.msgType, len(ch))
}

// 请求结构
type RequestStruct struct {
	Method string            `json:"method"`
	Path   string            `json:"path"`
	Params map[string]string `json:"params"`
}

// 响应结构
type responseStruct struct {
	Params map[string]string `json:"params"`
	Body   any               `json:"body"`
}

// 为响应体加param参数
func (c *Context) AddParam(key, value string) {
	if c.resp == nil {
		c.newresponse(nil)
	}
	c.resp.Params[key] = value
}

// 查询响应数据类型
func (c *Context) querySendType() int {
	return c.msgType
}

// 没有响应需求就不创建响应体
func (c *Context) newresponse(data any) {
	c.resp = &responseStruct{
		Params: make(map[string]string),
		Body:   data,
	}
}

// 解析请求参数和路径
func (c *Context) parse() error {
	c.req = &RequestStruct{}
	return c.BindJSON(c.req)
}

// 查询请求参数
func (c *Context) Query(key string) string {
	if c.req == nil {
		err := c.parse()
		if err != nil {
			log.Printf("Query parse failed err: %s\n", err.Error())
		}
	}
	return c.req.Params[key]
}

// 解析json请求到结构体
func (c *Context) BindJSON(obj any) error {
	return json.Unmarshal(c.message, &obj)
}

// 获取所属群组的其他成员id和连接
func (c *Context) getGroupMap() (conns map[WSId]*ConnStruct) {
	conns = c.mgr.groups[c.gname].conns
	return
}

// 组内广播信息【不包括自己】
// type = brodcast
func (c *Context) Brodcast(data any) {
	c.setBody(data)
	c.msgType = BRODCAST
	c.preprocress()

}

// 单独回复信息
func (c *Context) Response(data any) {
	c.setBody(data)
	c.msgType = SINGLE
	c.preprocress()
}

// 组内广播消息【包括自己】
func (c *Context) BrodcastMe(data any) {
	c.setBody(data)
	c.msgType = BRODCASTME
	c.preprocress()
}

// 用于延迟删除
func (c *Context) closeMe(data any) {
	c.setBody(data)
	c.msgType = CLOSE
	c.preprocress()
}

func (c *Context) setBody(data any) {
	if c.resp == nil {
		c.newresponse(data)
	} else {
		c.resp.Body = data
	}
}

func (c *Context) RootCache() cache.CacheI {
	return c.mgr.cache
}

// subcache belong to group
func (c *Context) Cache() cache.CacheI {
	return c.RootCache().SubCache(c.gname)
}

// subcache belong to conn
func (c *Context) ConnCache() cache.CacheI {
	return c.Cache().SubCache(c.IdString())
}

// id to string

func (c *Context) IdString() string {
	return strconv.Itoa(int(c.wsid))
}

func (c *Context) GroupName() string {
	return c.gname
}

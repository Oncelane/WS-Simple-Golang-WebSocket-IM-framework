package example_test

import (
	"aiosystem-backend/pkg/ws"
	"testing"

	"github.com/gin-gonic/gin"
)

// 1.最基本的用法，通过指定 :group 来标识此链接想要加入的群组
func TestBasic(t *testing.T) {
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

// 2.自定义ConnOpen, ConnClose等钩子函数
func TestCallBack(t *testing.T) {
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

// 3.Context的使用
// func TestContext(t *testing.T) {
// 	r := ws.Default()
// 	r.Use("post", "/talk", useContext)

// 	g := gin.Default()
// 	g.GET("/ws/:group", r.ConnectGIN)
// 	g.Run(":8080")
// }

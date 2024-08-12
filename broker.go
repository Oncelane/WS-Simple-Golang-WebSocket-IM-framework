package ws

import (
	"log"

	"github.com/gorilla/websocket"
)

type broker struct {
	id int
	ch chan *Context
}

func (g *Engine) initBroker() {
	g.brokers = make([]*broker, g.brokerNum)
	for i := range g.brokerNum {
		g.brokers[i] = &broker{
			id: i,
			ch: make(chan *Context),
		}
	}
}

func (g *Engine) runBroker() {
	for _, b := range g.brokers {
		go b.run(g)
	}
}

func (b *broker) run(g *Engine) {
	for ctx := range b.ch {
		// log.Println("new ctx", ctx.querySendType(), "ch left:", len(ch))
		// base on msgType, choose a way to send
		msgT := ctx.querySendType()
		switch msgT {
		case CLOSE:
			// handle groupClose callsBack
			groupStruct := g.groups[ctx.gname]
			log.Printf("Wsid [%d] 执行退群流程\n目前还有%d成员\n", ctx.wsid, len(groupStruct.conns))

			g.mu.Lock()
			delete(groupStruct.conns, ctx.wsid)
			g.mu.Unlock()
			ctx.ConnCache().Destroy()
			// close websocket connection
			err := ctx.conn.Close()
			if err != nil {
				log.Printf("[%s] [%d]this conn has already been close %s\n", ctx.gname, ctx.wsid, err.Error())
			}
			ok := len(groupStruct.conns) == 0
			if ok {
				log.Println("删除群组缓存")
				if g.onGroupClose != nil {
					g.onGroupClose(ctx) // groupClose callback
				}
				g.cache.SubCache(ctx.gname).Destroy()
			}
			continue
		// 单独回复
		case BRODCASTME, SINGLE:
			log.Printf("wsid[%d] <- msg (me)\n", ctx.wsid)
			g.mu.Lock()
			// log.Println("test ctx.conn", ctx.Conn)
			err := ctx.conn.WriteMessage(websocket.TextMessage, ctx.respMessage)
			ctx.conn.CloseHandler()
			if err != nil {
				// TODO: 后续改为心跳机制
				log.Println("err=", err)
				g.delete(ctx)
				continue
			}
			g.mu.Unlock()
			fallthrough
		// 组内广播
		case BRODCAST:
			if msgT == SINGLE {
				continue
			}
			g.mu.Lock()
			connMap := ctx.getGroupMap()
			if connMap == nil {
				log.Println("group not exist")
				g.mu.Unlock()
				continue
			}
			for id, connst := range connMap {
				// 消息广播只发给其他成员
				if id == ctx.wsid {
					continue
				}
				// err := conn.WriteJSON(c.response)
				log.Printf("wsid[%d] <- msg (brodcast)\n", id)
				err := connst.conn.WriteMessage(websocket.TextMessage, ctx.respMessage)
				if err != nil {
					// TODO: 后续改为心跳机制
					g.delete(ctx)
					continue
				}
			}
			g.mu.Unlock()
		default:
			log.Printf("wrong type of message: [%d]\n", msgT)
		}

	}

}

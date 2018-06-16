package hi

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	. "github.com/smartystreets/goconvey/convey"
)

func TestLobbyPlayer(t *testing.T) {
	Convey("LobbyPlayer", t, func() {
		p1 := &LobbyPlayer{
			Name:                "1234",
			MessagesToPlayer:    make(chan *MessageToPlayer),
			AddWSChannel:        make(chan *websocket.Conn),
			DisconnectWSChannel: make(chan *websocket.Conn),
			StopRoutines:        make(chan bool),
		}
		p2 := &LobbyPlayer{
			Name:         "1234",
			StopRoutines: make(chan bool),
		}
		Convey("AddSession", func() {
			Convey("will error if AddWSChannel is nil", func() {
				err := p2.AddSession(&websocket.Conn{})
				So(err, ShouldBeError)
				So(err.Error(), ShouldEqual, "AddWSChannel cannot be nil")
			})
			Convey("will timeout if channel blocked", func() {
				p2.AddWSChannel = make(chan *websocket.Conn)
				err := p2.AddSession(&websocket.Conn{})
				So(err, ShouldBeError)
				So(err.Error(), ShouldEqual, "could not add session for '1234' as channel was blocked")
			})
		})
		Convey("DisconnectSession", func() {
			Convey("will error if DisconnectWSChannel is nil", func() {
				err := p2.DisconnectSession(&websocket.Conn{})
				So(err, ShouldBeError)
				So(err.Error(), ShouldEqual, "DisconnectWSChannel cannot be nil")
			})
			Convey("will timeout if channel blocked", func() {
				p2.DisconnectWSChannel = make(chan *websocket.Conn)
				err := p2.DisconnectSession(&websocket.Conn{})
				So(err, ShouldBeError)
				So(err.Error(), ShouldEqual, "could not disconnect session for '1234' as channel was blocked")
			})
		})
		Convey("SessionHandler", func() {
			Convey("will make map on startup", func() {
				So(p1.sessions, ShouldBeNil)
				go p1.SessionHandler(55 * time.Second)
				defer close(p1.StopRoutines)
			})
			Convey("will shutdown when StopRoutines is closed", func() {
				stopped := make(chan bool)
				go func() {
					p2.SessionHandler(55 * time.Second)
					close(stopped)
				}()
				close(p2.StopRoutines)
				select {
				case <-stopped:
					So(true, ShouldBeTrue)
				case <-time.After(250 * time.Millisecond):
					So("didn't shut down", ShouldBeTrue)
				}
			})
			Convey("will add websocket to sessions from AddWSChannel channel", func() {
				go p1.SessionHandler(55 * time.Second)
				defer close(p1.StopRoutines)
				s1 := &websocket.Conn{}
				err := p1.AddSession(s1)
				So(err, ShouldBeNil)
				time.Sleep(250 * time.Millisecond)
				So(atomic.LoadInt32(&p1.atomicTotalSessions), ShouldEqual, p1.TotalSessions())
				Convey("will remove websocket from sessions from DisconnectWSChannel channel", func() {
					err := p1.DisconnectSession(s1)
					So(err, ShouldBeNil)
					time.Sleep(250 * time.Millisecond)
					So(atomic.LoadInt32(&p1.atomicTotalSessions), ShouldEqual, p1.TotalSessions())
				})
			})
		})
		Convey("MessageToPlayer", func() {
			Convey("errors without EventChannel", func() {
				err := p1.MessageToPlayer(&MessageToPlayer{})
				So(err, ShouldBeError)
				So(err.Error(), ShouldEqual, "missing EventChannel on MessageToPlayer")
			})
			Convey("errors without MessagesToPlayer chan", func() {
				err := p2.MessageToPlayer(&MessageToPlayer{})
				So(err, ShouldBeError)
				So(err.Error(), ShouldEqual, "missing MessagesToPlayer channel on player")
			})
			Convey("will time out if messages not picked up quick enough", func() {
				err := p1.MessageToPlayer(&MessageToPlayer{
					EventChannel: ChannelGlobal,
				})
				So(err, ShouldBeError)
				So(err.Error(), ShouldEqual, "timeout sending message(s)")
			})
			Convey("can place multiple message on the channel", func() {
				go func() {
					p1.MessageToPlayer(&MessageToPlayer{
						EventChannel: ChannelGlobal,
					}, &MessageToPlayer{
						EventChannel: ChannelGlobal,
					})
					close(p1.MessagesToPlayer)
				}()
				msgLen := 0
				for _ = range p1.MessagesToPlayer {
					msgLen++
				}
				So(msgLen, ShouldEqual, 2)
			})
		})
	})
}

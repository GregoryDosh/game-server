package hub

import (
	"testing"
	"time"

	hi "github.com/GregoryDosh/game-server/hub/hubinterfaces"
	"github.com/gorilla/websocket"
	. "github.com/smartystreets/goconvey/convey"
)

type testGame struct {
	GameName           string `json:"name"`
	startGameCalled    bool
	autoStartChan      chan bool
	autoStartCalled    bool
	endGameCalled      bool
	addPlayerCalled    bool
	removePlayerCalled bool
	playerEventCalled  bool
}

func (g *testGame) Name() string {
	return "Example Game Name Here"
}

func (g *testGame) Status() string {
	return "Created"
}

func (g *testGame) StartGame() error {
	g.startGameCalled = true
	return nil
}

func (g *testGame) EndGame() error {
	g.endGameCalled = true
	return nil
}

func (g *testGame) AddPlayer(p hi.PlayerInterface) (interface{}, error) {
	g.addPlayerCalled = true
	return nil, nil
}

func (g *testGame) RemovePlayer(p hi.PlayerInterface) error {
	g.removePlayerCalled = true
	return nil
}

func (g *testGame) PlayerEvent(p hi.PlayerInterface, e *hi.MessageFromPlayer) error {
	g.playerEventCalled = true
	return nil
}

func (g *testGame) AutoStart() {
	g.autoStartCalled = true
	g.autoStartChan <- true
}

func TestHub(t *testing.T) {
	Convey("Hub", t, func() {
		h := New()
		g1 := &testGame{
			GameName:      "Test",
			autoStartChan: make(chan bool, 0),
		}
		g2 := &testGame{
			GameName:      "OtherTest",
			autoStartChan: make(chan bool, 0),
		}
		p1 := &hi.LobbyPlayer{
			Name:                "P1",
			MessagesToPlayer:    make(chan *hi.MessageToPlayer, 256),
			StopRoutines:        make(chan bool),
			AddWSChannel:        make(chan *websocket.Conn),
			DisconnectWSChannel: make(chan *websocket.Conn),
		}
		p2 := &hi.LobbyPlayer{
			Name:                "P2",
			MessagesToPlayer:    make(chan *hi.MessageToPlayer, 256),
			StopRoutines:        make(chan bool),
			AddWSChannel:        make(chan *websocket.Conn),
			DisconnectWSChannel: make(chan *websocket.Conn),
		}
		p3 := &hi.LobbyPlayer{
			Name:                "P3",
			MessagesToPlayer:    make(chan *hi.MessageToPlayer, 256),
			StopRoutines:        make(chan bool),
			AddWSChannel:        make(chan *websocket.Conn),
			DisconnectWSChannel: make(chan *websocket.Conn),
		}
		p4 := &hi.LobbyPlayer{
			Name:                "P4",
			MessagesToPlayer:    make(chan *hi.MessageToPlayer, 256),
			StopRoutines:        make(chan bool),
			AddWSChannel:        make(chan *websocket.Conn),
			DisconnectWSChannel: make(chan *websocket.Conn),
		}
		ws1 := &websocket.Conn{}
		ws2 := &websocket.Conn{}
		go func() {
			p1.SessionHandler(55 * time.Second)
			p2.SessionHandler(55 * time.Second)
		}()
		defer func() {
			select {
			case p1.StopRoutines <- true:
			case <-time.After(25 * time.Millisecond):
			}
			select {
			case p2.StopRoutines <- true:
			case <-time.After(25 * time.Millisecond):
			}
		}()
		Convey("AddGame", func() {
			Convey("errors on nil game", func() {
				_, err := h.AddGame(nil)
				So(err, ShouldBeError)
				So(err.Error(), ShouldEqual, "invalid game created")
			})
			Convey("on success", func() {
				Convey("places game in Games map and returns UUID", func() {
					uuid, err := h.AddGame(g1)
					So(err, ShouldBeNil)
					So(len(h.games), ShouldEqual, 1)
					So(uuid, ShouldNotBeNil)
					So(g1, ShouldEqual, h.games[uuid])
				})
				Convey("sends a message to players in lobby with updated gamelist", func() {
					h.lobby["1234"] = p3
					_, err := h.AddGame(g1)
					time.Sleep(25 * time.Millisecond)
					So(err, ShouldBeNil)
					select {
					case msg := <-p3.MessagesToPlayer:
						So(msg.Type, ShouldEqual, "GAME_LIST")
						So(string(msg.Message), ShouldContainSubstring, `{"name":"Test"}`)
					case <-time.After(25 * time.Millisecond):
						So("Didn't get messages", ShouldBeTrue)
					}
				})
				Convey("calls AutoStart handler", func() {
					_, err := h.AddGame(g1)
					So(err, ShouldBeNil)
					<-g1.autoStartChan
					So(g1.autoStartCalled, ShouldBeTrue)
				})
			})
		})
		Convey("RemoveGame", func() {
			u1, _ := h.AddGame(g1)
			u2, _ := h.AddGame(g2)
			Convey("errors on nil game", func() {
				err := h.RemoveGame("")
				So(err, ShouldBeError)
				So(err.Error(), ShouldEqual, "UUID empty")
			})
			Convey("errors on missing game", func() {
				err := h.RemoveGame("1234")
				So(err, ShouldBeError)
				So(err.Error(), ShouldEqual, "could not find game with UUID '1234'")
			})
			Convey("on success", func() {
				Convey("calls EndGame on game for any cleanup", func() {
					err := h.RemoveGame(u1)
					So(err, ShouldBeNil)
					So(g1.endGameCalled, ShouldBeTrue)
				})
				Convey("removes game from Games map", func() {
					err := h.RemoveGame(u1)
					So(err, ShouldBeNil)
					So(len(h.games), ShouldEqual, 1)
				})
				Convey("sends a message to players in lobby with updated gamelist", func() {
					h.lobby["4321"] = p2
					err := h.RemoveGame(u2)
					So(err, ShouldBeNil)
					select {
					case msg := <-p2.MessagesToPlayer:
						So(msg.Type, ShouldEqual, "GAME_LIST")
						So(string(msg.Message), ShouldContainSubstring, `{"name":"Test"}`)
						So(string(msg.Message), ShouldNotContainSubstring, `{"name":"OtherTest"}`)
					case <-time.After(25 * time.Millisecond):
						So("Didn't get messages", ShouldBeTrue)
					}
				})
			})
		})
		Convey("ConnectSession", func() {
			Convey("for an empty websocket it will error", func() {
				p, err := h.ConnectSession("1234", nil, 55)
				So(p, ShouldBeNil)
				So(err, ShouldBeError)
				So(err.Error(), ShouldEqual, "missing websocket connection")
			})
			Convey("for a new/missing user should return a new PlayerInterface", func() {
				So(len(h.lobby), ShouldEqual, 0)
				p, err := h.ConnectSession("1234", ws1, 55)
				So(p, ShouldNotBeNil)
				So(err, ShouldBeNil)
				So(len(h.lobby), ShouldEqual, 1)
			})
			Convey("for an existing user should return existing PlayerInterface", func() {
				h.lobby["1234"] = p1
				So(len(h.lobby), ShouldEqual, 1)
				rp, err := h.ConnectSession("1234", ws2, 55)
				So(err, ShouldBeNil)
				So(rp, ShouldNotBeNil)
				So(len(h.lobby), ShouldEqual, 1)
				So(rp, ShouldEqual, p1)
			})
		})
		Convey("DisconnectSession", func() {
			Convey("will error if session not in connected sessions", func() {
				err := h.DisconnectSession("1234", &websocket.Conn{})
				So(err, ShouldBeError)
				So(err.Error(), ShouldEqual, "player with uuid '1234' not in lobby")
			})
			Convey("will error if session does not have websocket", func() {
				_, err := h.ConnectSession("1234", &websocket.Conn{}, 55)
				So(err, ShouldBeNil)
				err = h.DisconnectSession("1234", nil)
				So(err, ShouldBeError)
				So(err.Error(), ShouldEqual, "cannot disconnect nil websocket")
			})
			Convey("will succesfully remove session", func() {
				ws := &websocket.Conn{}
				_, err := h.ConnectSession("1234", ws, 55)
				So(err, ShouldBeNil)
				err = h.DisconnectSession("1234", ws)
				So(err, ShouldBeNil)
			})
		})
		Convey("UpdateGameList", func() {
			err := h.UpdateGameList()
			Convey("should not error", func() {
				So(err, ShouldBeNil)
			})
			Convey("sends a message to all players in lobby", func() {
				h.games["ABC"] = g1
				h.lobby["1234"] = p3
				h.lobby["4321"] = p4
				err := h.UpdateGameList()
				So(err, ShouldBeNil)
				select {
				case msg := <-p3.MessagesToPlayer:
					So(msg.Type, ShouldEqual, "GAME_LIST")
					So(string(msg.Message), ShouldContainSubstring, `{"name":"Test"}`)
				case <-time.After(25 * time.Millisecond):
					So("Didn't get messages", ShouldBeTrue)
				}
				select {
				case msg := <-p4.MessagesToPlayer:
					So(msg.Type, ShouldEqual, "GAME_LIST")
					So(string(msg.Message), ShouldContainSubstring, `{"name":"Test"}`)
				case <-time.After(25 * time.Millisecond):
					So("Didn't get messages", ShouldBeTrue)
				}
			})
			Convey("sends a message to only one player", func() {
				h.games["ABC"] = g1
				h.lobby["1234"] = p3
				h.lobby["4321"] = p4
				err := h.UpdateGameList(p3)
				So(err, ShouldBeNil)
				select {
				case msg := <-p3.MessagesToPlayer:
					So(msg.Type, ShouldEqual, "GAME_LIST")
					So(string(msg.Message), ShouldContainSubstring, `{"name":"Test"}`)
				case <-time.After(25 * time.Millisecond):
					So("Didn't get messages", ShouldBeTrue)
				}
				select {
				case <-p4.MessagesToPlayer:
					So("Shouldn't have gotten message", ShouldBeTrue)
				case <-time.After(25 * time.Millisecond):
				}
			})
		})
	})
}

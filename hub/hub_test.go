package hub

import (
	"testing"

	hi "github.com/GregoryDosh/game-server/hub/hubinterfaces"
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

func (g *testGame) PlayerEvent(p hi.PlayerInterface, e *hi.PlayerEvent) error {
	g.playerEventCalled = true
	return nil
}

func (g *testGame) AutoStart() {
	g.autoStartCalled = true
	g.autoStartChan <- true
}

type testPlayer struct {
	hi.PlayerInterface
	gotMessage bool
	messages   []*hi.MessageToPlayer
}

func (p *testPlayer) MessagePlayer(msgs ...*hi.MessageToPlayer) error {
	if len(msgs) > 0 {
		p.gotMessage = true
		for _, m := range msgs {
			p.messages = append(p.messages, m)
		}
	}
	return nil
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
		p1 := &testPlayer{}
		p2 := &testPlayer{}
		Convey("NewHub shouldn't error", func() {
			So(h, ShouldNotBeNil)
		})
		Convey("AddGame", func() {
			Convey("errors on nil game", func() {
				_, err := h.AddGame(nil)
				So(err, ShouldNotBeNil)
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
					h.lobby["1234"] = p1
					_, err := h.AddGame(g1)
					So(err, ShouldBeNil)
					So(p1.gotMessage, ShouldBeTrue)
					So(len(p1.messages), ShouldEqual, 1)
					So(p1.messages[0].Type, ShouldEqual, "GAME_LIST")
					So(p1.messages[0].Message, ShouldContainSubstring, `{"name":"Test"}`)
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
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "UUID empty")
			})
			Convey("errors on missing game", func() {
				err := h.RemoveGame("1234")
				So(err, ShouldNotBeNil)
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
					So(p2.gotMessage, ShouldBeTrue)
					So(len(p2.messages), ShouldEqual, 1)
					So(p2.messages[0].Type, ShouldEqual, "GAME_LIST")
					So(p2.messages[0].Message, ShouldContainSubstring, `{"name":"Test"}`)
					So(p2.messages[0].Message, ShouldNotContainSubstring, `{"name":"OtherTest"}`)
				})
			})
		})
	})
}

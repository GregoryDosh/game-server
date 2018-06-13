package moose

import (
	"fmt"
	"testing"
	"time"

	hi "github.com/GregoryDosh/game-server/hub/hubinterfaces"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGameSecretMoose(t *testing.T) {
	Convey("AddPlayer", t, func() {
		g := &GameSecretMoose{}
		Convey("returns PlayerSecretMoose type on success", func() {
			p, err := g.AddPlayer(&hi.LobbyPlayer{})
			switch v := p.(type) {
			case *PlayerSecretMoose:
				So(v, ShouldNotBeNil)
				So(err, ShouldBeNil)
				Convey("and adds PlayerSecretMoose to players", func() {
					So(g.Players[0], ShouldEqual, v)
				})
			default:
				fmt.Printf("unknown type '%T' encountered\n", v)
				t.Fail()
			}
		})
		Convey("errors when adding after game started", func() {
			for i := 1; i <= 5; i++ {
				p, _ := g.AddPlayer(&hi.LobbyPlayer{Name: fmt.Sprintf("P%d", i)})
				switch v := p.(type) {
				case *PlayerSecretMoose:
					v.IsReady = true
				}
			}
			g.StartGame()
			p, err := g.AddPlayer(&hi.LobbyPlayer{})
			So(p, ShouldBeNil)
			So(err, ShouldBeError)
			So(err.Error(), ShouldEqual, "cannot add player after game started")
		})
		Convey("errors when adding too many people to game", func() {
			for i := 1; i <= 10; i++ {
				p, _ := g.AddPlayer(&hi.LobbyPlayer{Name: fmt.Sprintf("P%d", i)})
				switch v := p.(type) {
				case *PlayerSecretMoose:
					v.IsReady = true
				}
			}
			p, err := g.AddPlayer(&hi.LobbyPlayer{})
			So(p, ShouldBeNil)
			So(err, ShouldBeError)
			So(err.Error(), ShouldEqual, "cannot add player as game is full")
		})
	})
	Convey("RemovePlayer", t, func() {
		g := &GameSecretMoose{}
		Convey("returns no error on success", func() {
			g.AddPlayer(&hi.LobbyPlayer{Name: "P1"})
			p, _ := g.AddPlayer(&hi.LobbyPlayer{Name: "P2"})
			g.AddPlayer(&hi.LobbyPlayer{Name: "P3"})
			So(len(g.Players), ShouldEqual, 3)
			switch v := p.(type) {
			case *PlayerSecretMoose:
				err := g.RemovePlayer(v.LobbyPlayer)
				So(err, ShouldBeNil)
				Convey("and removes from players", func() {
					So(len(g.Players), ShouldEqual, 2)
					So(g.Players, ShouldNotContain, v)
				})
			default:
				fmt.Printf("unknown type '%T' encountered\n", v)
				t.Fail()
			}
		})
		Convey("returns an error", func() {
			for i := 1; i <= 5; i++ {
				p, _ := g.AddPlayer(&hi.LobbyPlayer{Name: fmt.Sprintf("P%d", i)})
				switch v := p.(type) {
				case *PlayerSecretMoose:
					v.IsReady = true
				}
			}
			So(len(g.Players), ShouldEqual, 5)
			Convey("removing unknown player", func() {
				err := g.RemovePlayer(&hi.LobbyPlayer{})
				So(err, ShouldBeError)
				So(err.Error(), ShouldEqual, "could not remove unknown player")
			})
			Convey("when removing after game started", func() {
				g.StartGame()
				err := g.RemovePlayer(&hi.LobbyPlayer{})
				So(err, ShouldBeError)
				So(err.Error(), ShouldEqual, "cannot remove player after game started")
			})
		})
	})
	Convey("Name", t, func() {
		Convey("returns a non blank name if not specified", func() {
			g := &GameSecretMoose{}
			n := g.Name()
			So(n, ShouldEqual, DEFAULT_GAME_NAME)
		})
		Convey("returns set name if specified", func() {
			g := &GameSecretMoose{
				GameName: "Lunchtime Brawl",
			}
			n := g.Name()
			So(n, ShouldEqual, "Lunchtime Brawl")
		})
	})
	Convey("Status", t, func() {
		Convey("returns created if Status if not specified", func() {
			g := &GameSecretMoose{}
			n := g.Status()
			So(n, ShouldEqual, STATUS_CREATED)
		})
	})
	Convey("StartGame", t, func() {
		Convey("errors with not enough players in the game", func() {
			g := &GameSecretMoose{}
			err := g.StartGame()
			So(err, ShouldBeError)
			So(err.Error(), ShouldEqual, "not enough players to start")
		})
		Convey("errors with too many (>10) players in the game", func() {
			g := &GameSecretMoose{}
			g.Players = []*PlayerSecretMoose{{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}}
			err := g.StartGame()
			So(err, ShouldBeError)
			So(err.Error(), ShouldEqual, "too many players to start")
		})
		Convey("errors if all players are not ready", func() {
			g := &GameSecretMoose{}
			g.AddPlayer(&hi.LobbyPlayer{})
			g.AddPlayer(&hi.LobbyPlayer{})
			g.AddPlayer(&hi.LobbyPlayer{})
			g.AddPlayer(&hi.LobbyPlayer{})
			g.AddPlayer(&hi.LobbyPlayer{})
			err := g.StartGame()
			So(err, ShouldBeError)
			So(err.Error(), ShouldEqual, "players not ready to start")
		})
	})
	Convey("EndGame", t, func() {
		Convey("closes cancelAutostart channel if it exists", func() {
			g := &GameSecretMoose{
				cancelAutostart: make(chan bool, 0),
			}
			err := g.EndGame()
			n := g.Status()
			So(err, ShouldBeNil)
			So(n, ShouldEqual, STATUS_FINISHED)
		})
		Convey("does not panic if cancelAutostart does not exists", func() {
			g := &GameSecretMoose{}
			err := g.EndGame()
			n := g.Status()
			So(err, ShouldBeNil)
			So(n, ShouldEqual, STATUS_FINISHED)
		})
		Convey("changes Status to 'Finished'", func() {
			g := &GameSecretMoose{}
			err := g.EndGame()
			n := g.Status()
			So(err, ShouldBeNil)
			So(n, ShouldEqual, STATUS_FINISHED)
		})
	})
	Convey("AutoStart", t, func() {
		Convey("gracefully shuts down when receiving cancelAutostart", func() {
			g := &GameSecretMoose{
				cancelAutostart: make(chan bool, 0),
			}
			didShutdown := make(chan bool, 0)
			go func() {
				g.AutoStart()
				didShutdown <- true
			}()
			close(g.cancelAutostart)
			select {
			case <-didShutdown:
			case <-time.After(50 * time.Millisecond):
				fmt.Print("did not successfully shutdown")
				t.Fail()
			}
		})
		Convey("gracefully starts game when enough players and ready", func() {
			g := &GameSecretMoose{}
			for i := 1; i <= 5; i++ {
				p, _ := g.AddPlayer(&hi.LobbyPlayer{Name: fmt.Sprintf("P%d", i)})
				switch v := p.(type) {
				case *PlayerSecretMoose:
					v.IsReady = true
				}
			}
			didStartGame := make(chan bool, 0)
			go func() {
				g.AutoStart()
				didStartGame <- true
			}()
			select {
			case <-didStartGame:
				So(g.Status(), ShouldEqual, STATUS_STARTED)
			case <-time.After(50 * time.Millisecond):
				fmt.Print("did not successfully start game")
				t.Fail()
			}
		})
	})
	Convey("PlayerEvent", t, func() {
		Convey("returns error without player", func() {
			g := &GameSecretMoose{}
			err := g.PlayerEvent(nil, &hi.PlayerEvent{})
			So(err, ShouldBeError)
			So(err.Error(), ShouldEqual, "PlayerEvent require an active player")
		})
		Convey("returns error when player not in game", func() {
			g := &GameSecretMoose{}
			_, err := g.AddPlayer(&hi.LobbyPlayer{})
			err = g.PlayerEvent(&hi.LobbyPlayer{}, &hi.PlayerEvent{})
			So(err, ShouldBeError)
			So(err.Error(), ShouldEqual, "PlayerEvent require an active player")
		})
		Convey("Type ToggleReady", func() {
			g := &GameSecretMoose{}
			p1 := &hi.LobbyPlayer{}
			smp1, err := g.AddPlayer(p1)
			So(err, ShouldBeNil)
			g.PlayerEvent(p1, &hi.PlayerEvent{Type: "ToggleReady"})
			switch v := smp1.(type) {
			case *PlayerSecretMoose:
				So(v.IsReady, ShouldBeTrue)
			}
			g.PlayerEvent(p1, &hi.PlayerEvent{Type: "ToggleReady"})
			switch v := smp1.(type) {
			case *PlayerSecretMoose:
				So(v.IsReady, ShouldBeFalse)
			}
		})
	})
	Convey("lifecycle changes throughout game", t, func() {
		g := &GameSecretMoose{}
		n := g.Status()
		So(n, ShouldEqual, STATUS_CREATED)
		for i := 1; i <= 5; i++ {
			p, _ := g.AddPlayer(&hi.LobbyPlayer{})
			switch v := p.(type) {
			case *PlayerSecretMoose:
				v.IsReady = true
			}
		}
		g.StartGame()
		n = g.Status()
		So(n, ShouldEqual, STATUS_STARTED)
		g.EndGame()
		n = g.Status()
		So(n, ShouldEqual, STATUS_FINISHED)
	})
	Convey("team composition", t, func() {
		g := &GameSecretMoose{}
		for maxPlayers := 5; maxPlayers <= 10; maxPlayers++ {
			Convey(fmt.Sprintf("%d players", maxPlayers), func() {
				for i := 1; i <= maxPlayers; i++ {
					p, _ := g.AddPlayer(&hi.LobbyPlayer{Name: fmt.Sprintf("P%d", i)})
					switch v := p.(type) {
					case *PlayerSecretMoose:
						v.IsReady = true
					}
				}
				g.StartGame()
				switch maxPlayers {
				case 5:
					So(len(g.Players), ShouldEqual, 5)
					So(len(g.Fascists), ShouldEqual, 2)
					So(len(g.Liberals), ShouldEqual, 3)
				case 6:
					So(len(g.Players), ShouldEqual, 6)
					So(len(g.Fascists), ShouldEqual, 2)
					So(len(g.Liberals), ShouldEqual, 4)
				case 7:
					So(len(g.Players), ShouldEqual, 7)
					So(len(g.Fascists), ShouldEqual, 3)
					So(len(g.Liberals), ShouldEqual, 4)
				case 8:
					So(len(g.Players), ShouldEqual, 8)
					So(len(g.Fascists), ShouldEqual, 3)
					So(len(g.Liberals), ShouldEqual, 5)
				case 9:
					So(len(g.Players), ShouldEqual, 9)
					So(len(g.Fascists), ShouldEqual, 4)
					So(len(g.Liberals), ShouldEqual, 5)
				case 10:
					So(len(g.Players), ShouldEqual, 10)
					So(len(g.Fascists), ShouldEqual, 4)
					So(len(g.Liberals), ShouldEqual, 6)
				}
			})
		}
	})
}

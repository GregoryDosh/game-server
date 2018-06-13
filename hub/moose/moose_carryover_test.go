package moose

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	hi "github.com/GregoryDosh/game-server/hub/hubinterfaces"
	. "github.com/smartystreets/goconvey/convey"
)

func generatePlayerpool(s int, ready bool) []*hi.LobbyPlayer {
	playerPool := []*hi.LobbyPlayer{}
	for i := 1; i <= s; i++ {
		playerPool = append(playerPool, &hi.LobbyPlayer{Name: fmt.Sprintf("Player %2d", i)})
	}
	return playerPool
}

func randomTeam(t []*hi.LobbyPlayer, s int) []*hi.LobbyPlayer {
	returnPool := make([]*hi.LobbyPlayer, s)
	for i, randIndex := range rand.Perm(len(t))[:s] {
		returnPool[i] = t[randIndex]
	}
	return returnPool
}

func carryOverRed(t1, t2 []*PlayerSecretMoose) int {
	carryOver := 0
	for _, p := range t1 {
		for _, o := range t2 {
			if p.LobbyPlayer == o.LobbyPlayer {
				carryOver++
			}
		}
	}
	return carryOver
}

func TestSecretMooseCarryover(t *testing.T) {
	SkipConvey("testing carryover", t, func() {
		rand.Seed(time.Now().UTC().UnixNano())
		totalRuns := 10000
		for i := 5; i <= 15; i++ {
			totalPlayers := i
			playerPool := generatePlayerpool(totalPlayers, true)
			fmt.Printf("Total Player Pool: %2d\nTotal Runs %d\n", len(playerPool), totalRuns)
			for tp := 5; tp <= 10; tp++ {
				if tp > totalPlayers {
					continue
				}
				fmt.Printf("%2d Players ", tp)
				playersSelect := make(map[string]int)
				carryOver := 0
				carryOver1 := 0
				carryOver2 := 0
				carryOver3 := 0
				carryOver4 := 0
				prevFasc := []*PlayerSecretMoose{}
				for i := 1; i <= totalRuns; i++ {
					g := &GameSecretMoose{}
					for _, p := range randomTeam(playerPool, tp) {
						g.AddPlayer(p)
						g.PlayerEvent(p, &hi.MessageFromPlayer{Type: "ToggleReady"})
						playersSelect[p.Name]++
					}
					g.StartGame()
					switch val := carryOverRed(prevFasc, g.Fascists); val {
					case 4:
						carryOver++
						carryOver4++
					case 3:
						carryOver++
						carryOver3++
					case 2:
						carryOver++
						carryOver2++
					case 1:
						carryOver++
						carryOver1++
					case 0:
					default:
						panic(fmt.Sprintf("This shouldn't happen %d", val))
					}
					prevFasc = g.Fascists
				}
				fmt.Printf("distinct players selected: %2d, carryover total: %.3f%%, 1: %.3f%%, 2: %.3f%%, 3: %.3f%%, 4: %.3f%%\n", len(playersSelect), 100.0*float64(carryOver)/float64(totalRuns), 100.0*float64(carryOver1)/float64(totalRuns), 100.0*float64(carryOver2)/float64(totalRuns), 100.0*float64(carryOver3)/float64(totalRuns), 100.0*float64(carryOver4)/float64(totalRuns))
			}
		}
	})

}

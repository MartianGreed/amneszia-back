package game

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

var RevealTimeout = 2 * time.Second

type ConnectionBuf struct {
	timer       *time.Timer
	actions     []UserAction
	actionCount int
	x           int
	y           int
	x2          int
	y2          int
}

func (c *ConnectionBuf) appendAction(ua UserAction) {
	c.actionCount++
	c.actions = append(c.actions, ua)
}

func (c *ConnectionBuf) resetActions() {
	c.actions = []UserAction{}
}

type ConnectionPool struct {
	Connections map[*websocket.Conn]*ConnectionBuf
	sync.RWMutex
}

func NewConnectionPool() *ConnectionPool {
	return &ConnectionPool{
		Connections: map[*websocket.Conn]*ConnectionBuf{},
	}
}

type (
	UserAction struct {
		Event string
		X     int
		Y     int
	}
	UserRevealCardAction struct {
		Type string
		X    int
		Y    int
	}
	SystemHoverCardMessage struct {
		Event string
		X     int
		Y     int
	}
	SystemRevealCardMessage struct {
		Event     string
		Attribute Tile
		X         int
		Y         int
	}
	SystemHideCardMessage struct {
		Event     string
		Attribute Tile
		X         int
		Y         int
	}
)

// handle message type
// user.hover-card - make card at position floating
// user.reveal-card - send object with attribute at position after timeout of 2s send "system.hide-card"
//
//	if same user sends another request within 2s reveal the other card if two matches mark them as revealed and send picture
//
// system.hide-card - send object with false and position
func HandleMessage(ua UserAction, board *Board, ws *websocket.Conn, cp *ConnectionPool) error {
	switch ua.Event {
	case "user.hover-card":
		sendSystemHoverCard(cp, SystemHoverCardMessage{Event: "system.hover-card", X: ua.X, Y: ua.Y})
	case "user.leave-card":
		sendSystemHoverCard(cp, SystemHoverCardMessage{Event: "system.leave-card", X: ua.X, Y: ua.Y})
	case "user.reveal-card":
		incrementUserActionCounter(cp, ws, ua)
		_ = cp.Connections[ws]
		go sendSystemRevealCard(board, cp, ua)
		go hideCardAfterTimeout(board, cp, ws, ua)

		if len(cp.Connections[ws].actions) > 1 {
			prevActionIdx := len(cp.Connections[ws].actions) - 2
			prev := board.grid[cp.Connections[ws].actions[prevActionIdx].X][cp.Connections[ws].actions[prevActionIdx].Y]
			curr := board.grid[ua.X][ua.Y]

			if prev.Name == curr.Name {

				// do not hide the cards
				board.Revealed[cp.Connections[ws].actions[prevActionIdx].X][cp.Connections[ws].actions[prevActionIdx].Y].Revealed = true
				board.Revealed[cp.Connections[ws].actions[prevActionIdx].X][cp.Connections[ws].actions[prevActionIdx].Y].Attr = &board.grid[ua.X][ua.Y]
				board.Revealed[ua.X][ua.Y].Revealed = true
				board.Revealed[ua.X][ua.Y].Attr = &board.grid[cp.Connections[ws].actions[prevActionIdx].X][cp.Connections[ws].actions[prevActionIdx].Y]
				// execute match_tiles onchain

				cp.Connections[ws].resetActions()
			}
		}
	default: // unknown event
		return nil
	}

	return nil
}

func incrementUserActionCounter(cp *ConnectionPool, ws *websocket.Conn, ua UserAction) {
	if _, ok := cp.Connections[ws]; ok {
		cp.Connections[ws].appendAction(ua)
	}
}

func sendToConnectionPool(cp *ConnectionPool, t string, msg interface{}) {
	for connection := range cp.Connections {
		err := websocket.JSON.Send(connection, msg)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to send %s to %s", t, connection.Request().Header.Get("Sec-Websocket-Key")))
			// FIX: very odd but does not seem to have a case to handle those cases
			if strings.Contains(err.Error(), "write: broken pipe") {
				cp.Lock()
				delete(cp.Connections, connection)
				cp.Unlock()
			}
		}
	}
}

func sendSystemHoverCard(cp *ConnectionPool, a SystemHoverCardMessage) {
	rcm := SystemHoverCardMessage{
		Event: a.Event,
		X:     a.X,
		Y:     a.Y,
	}
	sendToConnectionPool(cp, "system.hover-card", rcm)
}

func sendSystemRevealCard(board *Board, cp *ConnectionPool, ua UserAction) {
	rcm := SystemRevealCardMessage{
		Event:     "system.reveal-card",
		Attribute: Tile{Attr: &board.grid[ua.X][ua.Y], Revealed: true},
		X:         ua.X,
		Y:         ua.Y,
	}
	sendToConnectionPool(cp, "system.reveal-card", rcm)
}

func sendSystemHideCard(b *Board, cp *ConnectionPool, action SystemHideCardMessage) {
	if b.Revealed[action.X][action.Y].Revealed {
		return
	}

	rcm := SystemHideCardMessage{
		Event:     "system.hide-card",
		Attribute: Tile{Attr: nil, Revealed: false},
		X:         action.X,
		Y:         action.Y,
	}
	sendToConnectionPool(cp, "system.hide-card", rcm)
}

func startTimer(b *Board, cp *ConnectionPool, ws *websocket.Conn, action UserAction) {
	cp.Lock()
	defer cp.Unlock()

	t := time.AfterFunc(RevealTimeout, func() {
		cp.Lock()
		defer cp.Unlock()
		cp.Connections[ws] = &ConnectionBuf{}
		sendSystemHideCard(b, cp, SystemHideCardMessage{Event: "system.hide-card", X: action.X, Y: action.Y})
	})
	cp.Connections[ws] = &ConnectionBuf{timer: t, x: action.X, y: action.Y}
}

func stopTimer(cp *ConnectionPool, ws *websocket.Conn) {
	if c, ok := cp.Connections[ws]; ok {
		cp.Lock()
		defer cp.Unlock()
		if c.timer != nil {
			c.timer.Stop()
		}
		cp.Connections[ws] = &ConnectionBuf{timer: nil}
	}
}

func resetTimer(b *Board, cp *ConnectionPool, ws *websocket.Conn, ua UserAction) {
	cp.Lock()
	defer cp.Unlock()
	c := cp.Connections[ws]
	if c.timer != nil {
		cp.Connections[ws].timer.Stop()
	}
	t := time.AfterFunc(RevealTimeout, func() {
		cp.Lock()
		defer cp.Unlock()

		sendSystemHideCard(b, cp, SystemHideCardMessage{Event: "system.hide-card", X: cp.Connections[ws].x, Y: cp.Connections[ws].y})
		sendSystemHideCard(b, cp, SystemHideCardMessage{Event: "system.hide-card", X: ua.X, Y: ua.Y})
		cp.Connections[ws] = &ConnectionBuf{timer: nil}
	})
	cp.Connections[ws] = &ConnectionBuf{timer: t, x2: ua.X, y2: ua.Y}
}

func hideCardAfterTimeout(b *Board, cp *ConnectionPool, ws *websocket.Conn, ua UserAction) {
	<-time.After(RevealTimeout)
	sendSystemHideCard(b, cp, SystemHideCardMessage{Event: "system.hide-card", X: ua.X, Y: ua.Y})
}

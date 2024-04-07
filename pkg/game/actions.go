package game

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

var RevealTimeout = 2 * time.Second

type ConnectionBuf struct {
	timer *time.Timer
	x     int
	y     int
	x2    int
	y2    int
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
		c := cp.Connections[ws]
		sendSystemRevealCard(board, cp, ua)
		if c.timer == nil {
			go startTimer(board, cp, ws, ua)
		} else {
			go resetTimer(board, cp, ws, ua)

			// check if tiles match
			prev := board.grid[c.x][c.y]
			curr := board.grid[ua.X][ua.Y]
			if prev.Name == curr.Name {
				go stopTimer(cp, ws)

				// do not hide the cards
				board.Revealed[c.x][c.y].Revealed = true
				board.Revealed[c.x][c.y].Attr = &prev
				board.Revealed[ua.X][ua.Y].Revealed = true
				board.Revealed[ua.X][ua.Y].Attr = &curr
				// execute match_tiles onchain

			}
		}
		cp.Connections[ws] = &ConnectionBuf{timer: nil}
	default: // unknown event
		return nil
	}

	return nil
}

func sendSystemHoverCard(cp *ConnectionPool, action SystemHoverCardMessage) {
	rcm := SystemHoverCardMessage{
		Event: action.Event,
		X:     action.X,
		Y:     action.Y,
	}
	for connection := range cp.Connections {
		err := websocket.JSON.Send(connection, rcm)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to send system.hover-card to %s", connection.Request().Header.Get("Sec-Websocket-Key")))
		}
	}
}

func sendSystemRevealCard(board *Board, cp *ConnectionPool, action UserAction) {
	rcm := SystemRevealCardMessage{
		Event:     "system.reveal-card",
		Attribute: Tile{Attr: &board.grid[action.X][action.Y], Revealed: true},
		X:         action.X,
		Y:         action.Y,
	}
	for connection := range cp.Connections {
		err := websocket.JSON.Send(connection, rcm)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to send system.reveal-card to %s", connection.Request().Header.Get("Sec-Websocket-Key")))
		}
	}
}

func sendSystemHideCard(b *Board, cp *ConnectionPool, action SystemHideCardMessage) {
	t := b.Revealed[action.X][action.Y]
	if t.Revealed {
		return
	}

	rcm := SystemHideCardMessage{
		Event:     "system.hide-card",
		Attribute: Tile{Attr: nil, Revealed: false},
		X:         action.X,
		Y:         action.Y,
	}
	for connection := range cp.Connections {
		err := websocket.JSON.Send(connection, rcm)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to send system.hover-card to %s", connection.Request().Header.Get("Sec-Websocket-Key")))
		}
	}
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

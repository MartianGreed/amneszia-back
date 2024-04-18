package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/MartianGreed/memo-backend/pkg/data"
	"github.com/MartianGreed/memo-backend/pkg/game"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/net/websocket"
)

var (
	connectionPool = game.NewConnectionPool()
	board          *game.Board
)

func hello(c echo.Context) error {
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()

		uuid := ws.Request().Header.Get("Sec-Websocket-Key")
		var userHello game.UserHello
		err := websocket.JSON.Receive(ws, &userHello)
		if err != nil && err.Error() != "EOF" {
			c.Logger().Error(err)
		}

		connectionPool.Lock()
		// execute join(contract_address: string, name: uuid) onchain to register user with wallet address
		connectionPool.Connections[ws] = &game.ConnectionBuf{Name: userHello.Name}

		defer func(connection *websocket.Conn) {
			connectionPool.Lock()
			delete(connectionPool.Connections, connection)
			connectionPool.Unlock()
		}(ws)

		connectionPool.Unlock()

		// on connection send the current board with revealed tiles
		// user.hover-card
		// user.reveal-card
		// board.game-has-finished

		slog.Info("connected " + uuid)
		jsonBoard, err := json.Marshal(board)
		if err != nil {
			c.Logger().Error(err)
		}

		// Write
		err = websocket.Message.Send(ws, string(jsonBoard))
		if err != nil {
			c.Logger().Error(err)
		}

		for {
			// Read
			var userAction game.UserAction
			err := websocket.JSON.Receive(ws, &userAction)
			if err != nil && err.Error() != "EOF" {
				c.Logger().Error(err)
			}

			err = game.HandleMessage(userAction, board, ws, connectionPool)
			if err != nil {
				c.Logger().Error(err)
			}
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// execute spawn() to create board onchain
	collection := data.LoadCollection()

	if board == nil {
		// fetch Tile collection
		// create board from fetched tiles
		board = game.CreateBoard(collection)
	}

	e.GET("/ws", hello)

	e.Logger.Fatal(e.Start(":8000"))

	go gracefulShutdown()
	forever := make(chan int)
	<-forever
}

func gracefulShutdown() {
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)
	signal.Notify(s, syscall.SIGTERM)
	go func() {
		<-s
		// clean up here
		os.Exit(0)
	}()
}

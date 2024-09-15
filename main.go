package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type   string   `json:"type"`
	Symbol string   `json:"symbol,omitempty"`
	Board  []string `json:"board,omitempty"`
	MyTurn bool     `json:"myTurn,omitempty"`
	Winner string   `json:"winner,omitempty"`
	Index  string   `json:"index,omitempty"`
}

type Game struct {
	Players [2]*websocket.Conn
	Board   []string
	Turn    int
	Mutex   sync.Mutex
}

var (
	tmpl     = template.Must(template.ParseFiles("templates/index.html"))
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	games = struct {
		game *Game
		sync.Mutex
	}{game: nil}
)

func main() {
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", handleWebSocket)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Println("Сервер запущен на :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	tmpl.Execute(w, nil)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Ошибка при апгрейде:", err)
		return
	}
	defer conn.Close()

	games.Lock()
	if games.game == nil {
		games.game = &Game{
			Board: make([]string, 9),
			Turn:  0,
		}
	}
	game := games.game
	if game.Players[0] == nil {
		game.Players[0] = conn
		sendStartMessage(conn, "X")
		if game.Players[1] != nil {
			sendStartMessage(game.Players[1], "O")
			sendUpdate(game)
		}
	} else if game.Players[1] == nil {
		game.Players[1] = conn
		sendStartMessage(conn, "O")
		sendUpdate(game)
	} else {
		// Игра уже заполнена
		conn.WriteJSON(Message{Type: "full"})
		games.Unlock()
		return
	}
	games.Unlock()

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("Ошибка чтения сообщения:", err)
			removePlayer(game, conn)
			break
		}
		handleMessage(game, conn, msg)
	}
}

func sendStartMessage(conn *websocket.Conn, symbol string) {
	msg := Message{
		Type:   "start",
		Symbol: symbol,
	}
	conn.WriteJSON(msg)
}

func sendUpdate(game *Game) {
	for i, player := range game.Players {
		if player == nil {
			continue
		}
		msg := Message{
			Type:   "update",
			Board:  game.Board,
			MyTurn: game.Turn == i,
		}
		player.WriteJSON(msg)
	}
}

func sendEnd(game *Game, winner string) {
	for _, player := range game.Players {
		if player == nil {
			continue
		}
		msg := Message{
			Type:   "end",
			Board:  game.Board,
			Winner: winner,
		}
		player.WriteJSON(msg)
	}
}

func handleMessage(game *Game, conn *websocket.Conn, msg Message) {
	if msg.Type == "move" {
		index := 0
		_, err := fmt.Sscanf(msg.Index, "%d", &index)
		if err != nil || index < 0 || index > 8 {
			return
		}

		game.Mutex.Lock()
		defer game.Mutex.Unlock()

		playerIndex := -1
		if game.Players[0] == conn {
			playerIndex = 0
		} else if game.Players[1] == conn {
			playerIndex = 1
		}
		if playerIndex != game.Turn {
			return // Не ваш ход
		}
		if game.Board[index] != "" {
			return // Ячейка уже занята
		}

		symbol := "X"
		if playerIndex == 1 {
			symbol = "O"
		}
		game.Board[index] = symbol

		if checkWin(game.Board, symbol) {
			sendEnd(game, symbol)
			game.Board = make([]string, 9)
			return
		} else if isDraw(game.Board) {
			sendEnd(game, "draw")
			game.Board = make([]string, 9)
			return
		}

		game.Turn = 1 - game.Turn
		sendUpdate(game)
	} else if msg.Type == "restart" {
		game.Mutex.Lock()
		game.Board = make([]string, 9)
		game.Turn = 0
		game.Mutex.Unlock()
		sendUpdate(game)
	}
}

func removePlayer(game *Game, conn *websocket.Conn) {
	game.Mutex.Lock()
	defer game.Mutex.Unlock()
	for i, player := range game.Players {
		if player == conn {
			game.Players[i] = nil
			break
		}
	}
}

func checkWin(board []string, symbol string) bool {
	wins := [8][3]int{
		{0, 1, 2}, {3, 4, 5}, {6, 7, 8}, // строки
		{0, 3, 6}, {1, 4, 7}, {2, 5, 8}, // столбцы
		{0, 4, 8}, {2, 4, 6}, // диагонали
	}
	for _, line := range wins {
		if board[line[0]] == symbol && board[line[1]] == symbol && board[line[2]] == symbol {
			return true
		}
	}
	return false
}

func isDraw(board []string) bool {
	for _, cell := range board {
		if cell == "" {
			return false
		}
	}
	return true
}

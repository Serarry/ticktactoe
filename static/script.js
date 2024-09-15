let ws;
let player = null;
let myTurn = false;
let board = ["", "", "", "", "", "", "", "", ""];
const statusDiv = document.getElementById("status");
const boardDiv = document.getElementById("board");
const restartButton = document.getElementById("restart");

// Инициализация игрового поля
function initBoard() {
    boardDiv.innerHTML = "";
    for (let i = 0; i < 9; i++) {
        const cell = document.createElement("div");
        cell.classList.add("cell");
        cell.dataset.index = i;
        cell.addEventListener("click", onCellClick);
        cell.innerText = board[i];
        boardDiv.appendChild(cell);
    }
}

// Обработчик клика по ячейке
function onCellClick(e) {
    const index = e.target.dataset.index;
    if (myTurn && board[index] === "") {
        ws.send(JSON.stringify({ type: "move", index: index }));
    }
}

// Установка статуса
function setStatus(message) {
    statusDiv.innerText = message;
}

// Обработчик рестарта
restartButton.addEventListener("click", () => {
    ws.send(JSON.stringify({ type: "restart" }));
});

// Установка символа игрока
function setPlayerSymbol(symbol) {
    player = symbol;
    if (player === "X") {
        myTurn = true;
        setStatus("Ваш ход (X)");
    } else {
        myTurn = false;
        setStatus("Ход соперника (X)");
    }
}

// Обновление игрового поля
function updateBoard(newBoard) {
    board = newBoard;
    initBoard();
}

// Обработка сообщений от сервера
function handleMessage(message) {
    const data = JSON.parse(message.data);
    switch (data.type) {
        case "start":
            setPlayerSymbol(data.symbol);
            break;
        case "update":
            updateBoard(data.board);
            myTurn = data.myTurn;
            setStatus(myTurn ? "Ваш ход (" + player + ")" : "Ход соперника (" + (player === "X" ? "O" : "X") + ")");
            break;
        case "end":
            updateBoard(data.board);
            if (data.winner) {
                setStatus(data.winner === "draw" ? "Ничья!" : (data.winner === player ? "Вы победили!" : "Вы проиграли!"));
            } else {
                setStatus("Игра окончена.");
            }
            restartButton.style.display = "block";
            break;
        default:
            console.log("Неизвестный тип сообщения:", data.type);
    }
}

// Подключение к WebSocket
function connect() {
    ws = new WebSocket("ws://" + window.location.host + "/ws");
    ws.onopen = () => {
        console.log("Подключено к серверу");
    };
    ws.onmessage = handleMessage;
    ws.onclose = () => {
        setStatus("Соединение разорвано.");
    };
}

initBoard();
connect();

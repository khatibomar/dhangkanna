type gameState = {
    guessedCharacter: [" ", "_"],
    incorrectGuesses: [],
    repeatedGuess: String,
    chancesLeft: Number,
    gameState: GameState,
    message: String
}

enum GameState {
    Start,
    Going,
    Won,
    Lost,
}

type socketEvent = {
    name: String,
    content: any
}

const characterDisplay = document.getElementById('characterDisplay');
const kannaImage : HTMLImageElement = document.getElementById('kannaImage') as HTMLImageElement;

const incorrectGuessesDisplay = document.getElementById('incorrectGuesses');
const chancesLeftDisplay = document.getElementById('chancesLeft');
const letterInput : HTMLInputElement = document.getElementById('letterInput') as HTMLInputElement;
const guessButton = document.getElementById('guessButton');

const gameMessage = document.getElementById('gameMessage');

const ws = new WebSocket(`ws://${location.host}/ws`);

ws.onopen = () => {
    console.log('WebSocket connection established.');
};

ws.onclose = (event) => {
    if (event.wasClean) {
        console.log(`WebSocket connection closed cleanly, code=${event.code}, reason=${event.reason}`);
    } else {
        console.error(`WebSocket connection closed unexpectedly. Code=${event.code}`);
    }
};

ws.onmessage = (event) => {
    const message : socketEvent = JSON.parse(event.data);
    console.log(message);
    switch (message.name) {
        case "game":
            console.log(message.content);
            const state: gameState = message.content;
            updateGame(state);
            break;
        case "notification":
            break;
    }
};

function updateGame(state: gameState) {
    switch (state.gameState) {
        case GameState.Won:
            showGameState(state.message, '#f4afca');
            kannaImage.src = 'static/kanna.gif';
            kannaImage.style.display = 'block';
            letterInput.disabled = true;
            guessButton.textContent = 'Restart';
            break;
        case GameState.Lost:
            showGameState(state.message, '#ff978d');
            letterInput.disabled = true;
            guessButton.textContent = 'Restart';
            kannaImage.src = 'static/sad_kanna.gif';
            kannaImage.style.display = 'block';
            break;
        case GameState.Going:
            showGameState(state.message, 'orange');
            break;
        case GameState.Start:
            resetGame();
            break;
        default:
            break;
    }

    chancesLeftDisplay.textContent = state.chancesLeft.toString();
    characterDisplay.textContent = state.guessedCharacter.join('');
    incorrectGuessesDisplay.textContent = state.incorrectGuesses.join(', ');
}

function focusInput() {
    letterInput.focus();
}

function resetGame() {
    kannaImage.style.display = 'none';
    gameMessage.textContent = '';
    letterInput.disabled = false;
    guessButton.textContent = 'Guess';
    letterInput.value = '';
    focusInput();
}

function notifyResetGame() {
    ws.send(JSON.stringify({ restart: true }));
}

function showGameState(message: String, color: String) {
    gameMessage.textContent = message.toString();
    gameMessage.style.color = color.toString();
}

function guessLetter() {
    const letter = letterInput.value.toLowerCase();

    ws.send(JSON.stringify({ letter: letter }));

    showGameState('', '');

    letterInput.value = '';
    focusInput();
}

letterInput.addEventListener('keydown', function(event) {
    if (event.key === 'Enter') {
        guessLetter();
    }
});

document.addEventListener('keydown', function(event) {
    if (event.key === 'r' || event.key === 'R') {
        if (document.activeElement !== letterInput) {
            event.preventDefault();
            notifyResetGame();
        }
    }
});

guessButton.addEventListener('click', function () {
    if(guessButton.textContent === "Restart") notifyResetGame();
    else guessLetter();
});

document.addEventListener('mouseleave', focusInput);
window.addEventListener('load', focusInput);
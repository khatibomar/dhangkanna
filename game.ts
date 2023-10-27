type gameState = {
    characterName: String,
    guessedCharacter: [" ", "_"],
    incorrectGuesses: [],
    repeatedGuess: String,
    chancesLeft: Number,
    gameWon: Boolean
}

const characterDisplay = document.getElementById('characterDisplay');
const kannaImage : HTMLImageElement = document.getElementById('kannaImage') as HTMLImageElement;

const incorrectGuessesDisplay = document.getElementById('incorrectGuesses');
const chancesLeftDisplay = document.getElementById('chancesLeft');
const letterInput : HTMLInputElement = document.getElementById('letterInput') as HTMLInputElement;
const guessButton = document.getElementById('guessButton');

const gameMessage = document.getElementById('gameMessage');

const ws = new WebSocket('ws://localhost:4000/ws');

ws.onopen = () => {
    console.log('WebSocket connection established.');
};

ws.onmessage = (event) => {
    const gameState : gameState = JSON.parse(event.data);
    console.log(gameState);
    updateGame(gameState);
};

function updateGame(state: gameState) {
    checkWin(state);
    checkLoss(state);

    chancesLeftDisplay.textContent = state.chancesLeft.toString();
    characterDisplay.textContent = state.guessedCharacter.join('');
    incorrectGuessesDisplay.textContent = state.incorrectGuesses.join(', ');

    if(state.repeatedGuess) {
        showGameState(`You already picked ${state.repeatedGuess}.`, 'orange');
    }
}

function focusInput() {
    letterInput.focus();
}

function checkWin(state: gameState) {
    if (state.guessedCharacter.join('') === state.characterName) {
        showGameState('Congratulations! You win!', '#f4afca');
        kannaImage.src = 'static/kanna.gif';
        kannaImage.style.display = 'block';

        letterInput.disabled = true;
        guessButton.textContent = 'Restart';
    }
}

function checkLoss(state: gameState) {
    if (state.chancesLeft === 0 && !state.gameWon) {
        showGameState('You lose! The character was: ' + state.characterName, '#ff978d');
        letterInput.disabled = true;
        guessButton.textContent = 'Restart';
        kannaImage.src = 'static/sad_kanna.gif';
        kannaImage.style.display = 'block';
    }
}

function resetGame() {
    kannaImage.style.display = 'none';
    gameMessage.textContent = '';
    letterInput.disabled = false;
    guessButton.textContent = 'Guess';
    letterInput.value = '';

    ws.send(JSON.stringify({ restart: true }));
}

function showGameState(message, color) {
    gameMessage.textContent = message;
    gameMessage.style.color = color;
}

function guessLetter() {
    const letter = letterInput.value.toLowerCase();

    if (letter.length !== 1 || !/^[a-z]$/.test(letter)) {
        showGameState('Please enter a valid single letter.', 'orange');
        letterInput.value = '';
        return;
    }

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
            resetGame();
            focusInput();
        }
    }
});

guessButton.addEventListener('click', function () {
    if (guessButton.textContent === 'Restart') {
        resetGame();
    } else {
        guessLetter();
    }
});
document.addEventListener('mouseleave', focusInput);
window.addEventListener('load', focusInput);
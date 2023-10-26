const characterName = "kanna kamui";
const characterDisplay = document.getElementById('characterDisplay');
const kannaImage = document.getElementById('kannaImage');

const characterNameArray = characterName.split('');
let guessedCharacter = characterNameArray.map(char => (char === ' ' ? ' ' : '_'));
let incorrectGuesses = [];
let chancesLeft = 6;
let gameWon = false;

const incorrectGuessesDisplay = document.getElementById('incorrectGuesses');
const chancesLeftDisplay = document.getElementById('chancesLeft');
const letterInput = document.getElementById('letterInput');
const guessButton = document.getElementById('guessButton');

const gameMessage = document.getElementById('gameMessage');

function focusInput() {
    letterInput.focus();
}

function updateCharacterDisplay() {
    characterDisplay.textContent = guessedCharacter.join('');
}

function checkWin() {
    if (guessedCharacter.join('') === characterName) {
        showGameState('Congratulations! You win!', '#f4afca');
        kannaImage.src = 'static/kanna.gif';
        kannaImage.style.display = 'block';
        gameWon = true;
        letterInput.disabled = true;
        guessButton.textContent = 'Restart';
    }
}

function checkLoss() {
    if (chancesLeft === 0 && !gameWon) {
        showGameState('You lose! The character was: ' + characterName, '#ff978d');
        letterInput.disabled = true;
        guessButton.textContent = 'Restart';
        kannaImage.src = 'static/sad_kanna.gif';
        kannaImage.style.display = 'block';
    }
}

function resetGame() {
    guessedCharacter = characterNameArray.map(char => (char === ' ' ? ' ' : '_'));
    incorrectGuesses = [];
    chancesLeft = 6;
    gameWon = false;
    updateCharacterDisplay();
    incorrectGuessesDisplay.textContent = '';
    chancesLeftDisplay.textContent = chancesLeft;
    kannaImage.style.display = 'none';
    gameMessage.textContent = '';
    letterInput.disabled = false;
    guessButton.textContent = 'Guess';
    letterInput.value = '';
}

function showGameState(message, color) {
    gameMessage.textContent = message;
    gameMessage.style.color = color;
}

function guessLetter() {
    if (gameWon) {
        resetGame();
        return;
    }

    const letter = letterInput.value.toLowerCase();

    if (letter.length !== 1 || !/^[a-z]$/.test(letter)) {
        showGameState('Please enter a valid single letter or space.', 'orange');
        letterInput.value = '';
        return;
    }

    showGameState('', '');
    if (incorrectGuesses.includes(letter) || guessedCharacter.includes(letter)) {
        showGameState('You already guessed that letter.', 'orange');
    } else if (characterNameArray.includes(letter)) {
        for (let i = 0; i < characterNameArray.length; i++) {
            if (characterNameArray[i] === letter) {
                guessedCharacter[i] = letter;
            }
        }
        updateCharacterDisplay();
        checkWin();
    } else {
        if (!incorrectGuesses.includes(letter)) {
            incorrectGuesses.push(letter);
            incorrectGuessesDisplay.textContent = incorrectGuesses.join(', ');
            chancesLeft--;
            chancesLeftDisplay.textContent = chancesLeft;
            checkLoss();
        }
    }

    letterInput.value = '';
    focusInput();
}

letterInput.addEventListener('keydown', function(event) {
    if (event.key === 'Enter') {
        guessLetter();
    }
});

document.addEventListener('keydown', function(event) {
    if (gameWon || chancesLeft === 0) {
        if (event.key === 'r' || event.key === 'R') {
            if (document.activeElement !== letterInput) {
                event.preventDefault();
                resetGame();
                focusInput();
            }
        }
    }
});

updateCharacterDisplay();
guessButton.addEventListener('click', guessLetter);
document.addEventListener('mouseleave', focusInput);
window.addEventListener('load', focusInput);
package state

const CharacterName = "kanna kamui"

type State struct {
	CharacterName    string   `json:"characterName"`
	GuessedCharacter []string `json:"guessedCharacter"`
	IncorrectGuesses []string `json:"incorrectGuesses"`
	RepeatedGuess    string   `json:"repeatedGuess"`
	ChancesLeft      int      `json:"chancesLeft"`
	GameWon          bool     `json:"gameWon"`
	NewGame          bool     `json:"newGame"`
}

func New() *State {
	return &State{
		CharacterName:    CharacterName,
		GuessedCharacter: InitializeGuessedCharacter(CharacterName),
		IncorrectGuesses: make([]string, 0),
		ChancesLeft:      6,
	}
}

func InitializeGuessedCharacter(characterName string) []string {
	guessedCharacter := make([]string, len(characterName))
	for i, char := range characterName {
		if char == ' ' {
			guessedCharacter[i] = " "
		} else {
			guessedCharacter[i] = "_"
		}
	}
	return guessedCharacter
}

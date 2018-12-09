package lib

import (
	"fmt"
	"regexp"

	"github.com/draganshadow/trello"
)

type Trello struct {
	appKey                   string
	token                    string
	resultBoardShortLink     string
	incomingResultColumnName string
	resultBoard              *trello.Board
	incomingResultList       *trello.List
	boardCards               []*trello.Card
}

func (t *Trello) Init() {
	t.getTrelloBoard()
	t.getTrelloBoardList()
	t.clearList()
}

func (t *Trello) getTrelloBoard() {
	client := trello.NewClient(t.appKey, t.token)
	var err error
	t.resultBoard, err = client.GetBoard(t.resultBoardShortLink, trello.Defaults())
	if err != nil {
		// Handle error
	}
}

func (t *Trello) getTrelloBoardList() {
	fmt.Printf("Get Trello List : %s \n", t.incomingResultColumnName)
	fmt.Println("Result Board", t.resultBoard.Name)
	resultBoardLists, err := t.resultBoard.GetLists(trello.Defaults())
	if err != nil {
		// Handle error
	}
	for _, rblist := range resultBoardLists {
		if rblist.Name == t.incomingResultColumnName {
			t.incomingResultList = rblist
			break
		}
	}
}

func (t *Trello) getCards() {
	var err error
	t.boardCards, err = t.resultBoard.GetCards(trello.Defaults())

	fmt.Printf("Board contain %d cards\n", len(t.boardCards))
	if err != nil {
		// Handle error
	}
}
func (t *Trello) clearList() {
	rbcards, err := t.incomingResultList.GetCards(trello.Defaults())
	if err != nil {
		// Handle error
	}

	for _, card := range rbcards {
		err := card.Delete(trello.Defaults())
		if err != nil {
			// Handle error
		}
	}
}

func (t *Trello) findCardByUID(UID string) (*trello.Card, bool) {
	for _, c := range t.boardCards {

		re := regexp.MustCompile("UID\\s*:\\s*" + UID)
		if re.FindString(c.Desc) != "" {
			return c, true
		}
	}
	return nil, false
}

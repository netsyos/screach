package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/draganshadow/trello"
	"github.com/gorilla/mux"
	"github.com/tebeka/selenium"
)

type Config struct {
	SeleniumHost string   `json:"seleniumHost"`
	SeleniumPort string   `json:"seleniumPort"`
	AppKey       string   `json:"appKey"`
	Token        string   `json:"token"`
	Member       string   `json:"member"`
	Searchs      []Search `json:"searchs"`
}

type Search struct {
	StartURL                 string  `json:"startURL"`
	ResultBoardShortLink     string  `json:"resultBoardShortLink"`
	IncomingResultColumnName string  `json:"incomingResultColumnName"`
	Scraps                   []Scrap `json:"scraps"`
}

type Scrap struct {
	CSSSelector string  `json:"cssSelector"`
	CardElement string  `json:"cardElement"`
	Prepend     string  `json:"prepend"`
	DomField    string  `json:"domField"`
	Follow      bool    `json:"follow"`
	Scraps      []Scrap `json:"scraps"`
}

type ScrapResult struct {
	CardElement  string
	Text         string
	ScrapResults []ScrapResult
}

func readConfig() Config {
	configFile := os.Getenv("SCREACH_CONFIG")
	if "" == configFile {
		configFile = "config.json"
	}
	fmt.Println("SCREACH_CONFIG: ", configFile)
	// Open our jsonFile
	jsonFile, err := os.Open(configFile)

	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Successfully Opened config.json")
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	var config Config
	json.Unmarshal(byteValue, &config)
	fmt.Printf("config : %+v\n", config)
	return config
}

func main() {
	var err error
	config := readConfig()

	r := mux.NewRouter()
	r.HandleFunc("/status/{service}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		service := vars["service"]

		fmt.Fprintf(w, "You've requested the status : %s\n", service)
	})

	// http.ListenAndServe(":80", r)

	// Connect to the WebDriver instance running locally.
	// f := firefox.Capabilities{}
	// f.Binary = "vendor/github.com/tebeka/selenium/vendor/firefox-nightly/firefox"
	// f.Log = &firefox.Log{
	// 	Level: firefox.Error,
	// }
	caps := selenium.Capabilities{
		"browserName": "firefox",
		// "moz:firefoxOptions": f,
	}
	var wd selenium.WebDriver
	for {
		wd, err = selenium.NewRemote(caps, fmt.Sprintf("http://%s:%s/wd/hub", config.SeleniumHost, config.SeleniumPort))

		if err != nil {
			fmt.Println("Wait Selenium to be ready")
			time.Sleep(10 * time.Second)
		} else {
			break
		}
	}
	// if err != nil {
	// 	panic(err)
	// }
	defer wd.Quit()

	fmt.Println("Process Search List")
	for _, s := range config.Searchs {
		doSearch(wd, config, s)
	}

}

func getTrelloBoardList(appKey string, token string, resultBoardShortLink string, incomingResultColumnName string) *trello.List {
	fmt.Printf("Get Trello List : %s \n", incomingResultColumnName)
	client := trello.NewClient(appKey, token)

	resultBoard, err := client.GetBoard(resultBoardShortLink, trello.Defaults())
	if err != nil {
		// Handle error
	}
	fmt.Println("Result Board", resultBoard.Name)
	resultBoardLists, err := resultBoard.GetLists(trello.Defaults())
	var incomingResultList *trello.List
	if err != nil {
		// Handle error
	}
	for _, rblist := range resultBoardLists {
		if rblist.Name == incomingResultColumnName {
			incomingResultList = rblist
			rbcards, err := rblist.GetCards(trello.Defaults())
			if err != nil {
				// Handle error
			}

			for _, card := range rbcards {
				err := card.Delete(trello.Defaults())
				if err != nil {
					// Handle error
				}
			}
			break
		}
	}
	return incomingResultList
}

func exportResultToTrelloList(result ScrapResult, incomingResultList *trello.List) {
	fmt.Printf("exportResultToTrelloList\n")
	card := resultToCard(result)
	err := incomingResultList.AddCard(&card, trello.Defaults())
	if err != nil {
		//Handle
	}
}

func resultToCard(result ScrapResult) trello.Card {
	fmt.Printf("resultToCard\n")

	card := trello.Card{
		Name: "",
		Desc: "",
	}
	fmt.Printf("result : %s - %s\n", result.CardElement, result.Text)
	if result.CardElement == "name" {
		card.Name = result.Text
	}
	if result.CardElement == "description" {
		card.Desc = result.Text
	}
	if result.CardElement == "attachment" {
		card.Desc += "Attachment : " + result.Text
		attachment := trello.Attachment{
			URL: result.Text,
		}
		card.Attachments = append(card.Attachments, &attachment)
	}

	for _, r := range result.ScrapResults {
		subCard := resultToCard(r)
		if subCard.Name != "" {
			if card.Name != "" {
				card.Name += " / "
			}
			card.Name += subCard.Name
		}

		if subCard.Desc != "" {
			if card.Desc != "" {
				card.Desc += "\n"
			}
			card.Desc += subCard.Desc
		}

		if len(subCard.Attachments) > 0 {
			card.Attachments = append(card.Attachments, subCard.Attachments...)
		}
	}
	return card
}

func doSearch(wd selenium.WebDriver, config Config, search Search) {
	fmt.Printf("Do Search\n")
	incomingResultList := getTrelloBoardList(config.AppKey, config.Token, search.ResultBoardShortLink, search.IncomingResultColumnName)
	if err := wd.Get(search.StartURL); err != nil {
		panic(err)
	}
	for _, scrap := range search.Scraps {
		result := doScrap(wd, nil, scrap)
		fmt.Printf("doScrap result : %+v \n", result)
		for _, r := range result.ScrapResults {
			exportResultToTrelloList(r, incomingResultList)
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func doScrap(wd selenium.WebDriver, parent selenium.WebElement, scrap Scrap) ScrapResult {
	fmt.Printf("Do Scrap\n")

	rand.Seed(time.Now().UnixNano())
	var items []selenium.WebElement
	var err error
	if parent == nil {
		fmt.Printf(" - Selector : %s \n", scrap.CSSSelector)
		items, err = wd.FindElements(selenium.ByCSSSelector, scrap.CSSSelector)
	} else {
		fmt.Printf(" - SubSelector : %s \n", scrap.CSSSelector)
		items, err = parent.FindElements(selenium.ByCSSSelector, scrap.CSSSelector)
	}

	if err != nil {
		panic(err)
	}

	var result ScrapResult
	for _, item := range items {
		var output string

		var itemResult ScrapResult

		if scrap.DomField != "" {
			fmt.Printf("scrap dom field : %s \n", scrap.DomField)
			output, err = item.GetAttribute(scrap.DomField)
		} else {
			output, err = item.Text()
		}

		if err != nil {
			panic(err)
		}

		fmt.Printf("output : %s \n", output)

		if scrap.CardElement != "" {
			if scrap.Prepend != "" {
				itemResult.Text += scrap.Prepend
			}
			itemResult.Text += output
			itemResult.CardElement = scrap.CardElement
		}

		if scrap.Follow {
			fmt.Printf("follow : %s \n", itemResult.Text)
			time.Sleep(time.Duration(1+rand.Intn(100)) * time.Second)
			err = wd.Get(itemResult.Text)
			if err != nil {
				panic(err)
			}
			for _, subScrap := range scrap.Scraps {
				subResult := doScrap(wd, nil, subScrap)
				itemResult.ScrapResults = append(itemResult.ScrapResults, subResult)
			}
			err = wd.Back()
			if err != nil {
				panic(err)
			}
		} else {
			for _, subScrap := range scrap.Scraps {
				subResult := doScrap(wd, item, subScrap)
				itemResult.ScrapResults = append(itemResult.ScrapResults, subResult)
			}
		}
		result.ScrapResults = append(result.ScrapResults, itemResult)

		fmt.Printf("Force exit \n")
		break
	}

	return result
}

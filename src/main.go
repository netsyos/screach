package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"time"

	"encoding/hex"

	"github.com/draganshadow/trello"
	"github.com/gorilla/mux"
	"github.com/tebeka/selenium"
)

type Config struct {
	SeleniumHost           string   `json:"seleniumHost"`
	SeleniumPort           string   `json:"seleniumPort"`
	TestMode               bool     `json:"testMode"`
	RandomSleepBeforeStart int      `json:"randomSleepBeforeStart"`
	AppKey                 string   `json:"appKey"`
	Token                  string   `json:"token"`
	Member                 string   `json:"member"`
	Templates              []Search `json:"templates"`
	Searchs                []Search `json:"searchs"`
}

type Search struct {
	Template                 string  `json:"template"`
	StartURL                 string  `json:"startURL"`
	ResultBoardShortLink     string  `json:"resultBoardShortLink"`
	IncomingResultColumnName string  `json:"incomingResultColumnName"`
	Paginator                Scrap   `json:"paginator"`
	Scraps                   []Scrap `json:"scraps"`
}

type Scrap struct {
	Name        string  `json:"name"`
	FindBy      string  `json:"findBy"`
	Selector    string  `json:"selector"`
	CardElement string  `json:"cardElement"`
	Prepend     string  `json:"prepend"`
	DomField    string  `json:"domField"`
	Do          string  `json:"do"`
	Follow      bool    `json:"follow"`
	Scraps      []Scrap `json:"scraps"`
}

type ScrapResult struct {
	UID          string
	CardElement  string
	Name         string
	Text         string
	Follow       bool
	Scrap        Scrap
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

	rand.Seed(time.Now().UnixNano())
	if config.RandomSleepBeforeStart > 0 {
		secToWait := rand.Intn(config.RandomSleepBeforeStart)
		fmt.Printf("Let's wait before start : %d\n", secToWait)
		time.Sleep(time.Duration(secToWait) * time.Second)
	}
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

func getTrelloBoard(appKey string, token string, resultBoardShortLink string) *trello.Board {
	client := trello.NewClient(appKey, token)

	resultBoard, err := client.GetBoard(resultBoardShortLink, trello.Defaults())
	if err != nil {
		// Handle error
	}
	return resultBoard
}
func getTrelloBoardList(resultBoard *trello.Board, incomingResultColumnName string) *trello.List {
	fmt.Printf("Get Trello List : %s \n", incomingResultColumnName)

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
	resultBoard.GetCards(trello.Defaults())
	return incomingResultList
}

func exportResultToTrelloList(result ScrapResult, resultBoard *trello.Board, incomingResultList *trello.List) {
	// fmt.Printf("exportResultToTrelloList\n")
	card := resultToCard(result)
	err := incomingResultList.AddCard(&card, trello.Defaults())
	for i := len(card.Attachments) - 1; i >= 0; i-- {
		fmt.Printf("Add attach\n")
		a := card.Attachments[i]
		card.AttachURL(a.Name, a.URL)
	}

	if err != nil {
		//Handle
	}
}
func GetMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}
func resultToCard(result ScrapResult) trello.Card {
	//fmt.Printf("resultToCard\n")

	card := trello.Card{
		Name: "",
		Desc: "",
	}
	// fmt.Printf("result : %s - %s\n", result.CardElement, result.Text)
	if result.Text != "" {
		if result.CardElement == "name" {
			card.Name = result.Text
		}
		if result.CardElement == "description" {
			card.Desc = result.Text
		}
		if result.CardElement == "attachment" {
			card.Desc += "Attachment : " + result.Text
			attachment := trello.Attachment{
				Name: result.Name,
				URL:  result.Text,
			}
			card.Attachments = append(card.Attachments, &attachment)
		}
		if result.Name == "UID" {
			card.Desc += "UID:" + GetMD5Hash(result.Text)
		}
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
				card.Desc += "\n\n\n"
			}
			card.Desc += subCard.Desc
		}

		if len(subCard.Attachments) > 0 {
			card.Attachments = append(card.Attachments, subCard.Attachments...)
		}
	}
	return card
}

func findCardByUID(cardList []*trello.Card, UID string) (*trello.Card, bool) {
	for _, c := range cardList {

		re := regexp.MustCompile("UID:" + UID)
		if re.FindString(c.Desc) != "" {
			return c, true
		}
	}
	return nil, false
}

func doSearch(wd selenium.WebDriver, config Config, search Search) {
	fmt.Printf("Do Search\n")
	var searchTemplate Search
	if search.Template != "" {
		for _, template := range config.Templates {
			if template.Template == search.Template {
				searchTemplate = template
				break
			}
		}
		if search.StartURL != "" {
			searchTemplate.StartURL = search.StartURL
		}
		if search.ResultBoardShortLink != "" {
			searchTemplate.ResultBoardShortLink = search.ResultBoardShortLink
		}
		if search.IncomingResultColumnName != "" {
			searchTemplate.IncomingResultColumnName = search.IncomingResultColumnName
		}
		search = searchTemplate
	}
	resultBoard := getTrelloBoard(config.AppKey, config.Token, search.ResultBoardShortLink)
	incomingResultList := getTrelloBoardList(resultBoard, search.IncomingResultColumnName)

	existingCards, err := resultBoard.GetCards(trello.Defaults())
	fmt.Printf("Board contain %d cards\n", len(existingCards))
	if err != nil {
		// Handle error
	}
	mainURL := search.StartURL
	paginateNext := true
	page := 1

	for paginateNext {

		fmt.Printf("Search page %d\n", page)
		for _, scrap := range search.Scraps {
			if err := wd.Get(mainURL); err != nil {
				panic(err)
			}
			result := doScrap(wd, nil, scrap, config.TestMode)
			// fmt.Printf("doScrap result : %+v \n", result)
			for _, r := range result.ScrapResults {
				_, found := findCardByUID(existingCards, r.UID)
				if !found {

					fmt.Printf("No card with UID %s card will be added\n", r.UID)
					exportResultToTrelloList(r, resultBoard, incomingResultList)
					time.Sleep(100 * time.Millisecond)
				} else {

					fmt.Printf("Ignoring existing card with UID %s\n", r.UID)
				}
			}
		}

		if err := wd.Get(mainURL); err != nil {
			panic(err)
		}
		paginator := doScrap(wd, nil, search.Paginator, config.TestMode)
		if len(paginator.ScrapResults) == 1 {
			mainURL = paginator.ScrapResults[0].Text
			fmt.Printf("next page %s\n", mainURL)
			page++
		} else {
			paginateNext = false
		}
		if config.TestMode {
			paginateNext = false
		}
	}
}

func doScrap(wd selenium.WebDriver, parent selenium.WebElement, scrap Scrap, testMode bool) ScrapResult {
	fmt.Printf("Do Scrap\n")

	rand.Seed(time.Now().UnixNano())
	var items []selenium.WebElement
	var err error
	findBy := selenium.ByCSSSelector
	switch scrap.FindBy {
	case "xpath":
		findBy = selenium.ByXPATH
	default:
		findBy = selenium.ByCSSSelector
	}
	if parent == nil {
		fmt.Printf(" - Selector (%s) : %s \n", findBy, scrap.Selector)
		items, err = wd.FindElements(findBy, scrap.Selector)
	} else {
		fmt.Printf(" - SubSelector (%s) : %s \n", findBy, scrap.Selector)
		items, err = parent.FindElements(findBy, scrap.Selector)
	}

	if err != nil {
		panic(err)
	}

	fmt.Printf("Items found : %d \n", len(items))
	var result ScrapResult
	result.Name = scrap.Name
	if scrap.Follow {
		result.Follow = true
		result.Scrap = scrap
		fmt.Println("Enable follow flag")
	}
	followSub := false
	for i, item := range items {
		fmt.Printf("Process item : %d \n", i)
		var output string
		if scrap.Do != "" {
			switch scrap.Do {
			case "click":
				fmt.Printf("Do Click \n")
				time.Sleep(time.Duration(100+rand.Intn(2000)) * time.Millisecond)
				item.Click()
				time.Sleep(3 * time.Second)
			}
		}
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

		if output != "" {

			var itemResult ScrapResult
			itemResult.Name = scrap.Name

			if scrap.Prepend != "" {
				itemResult.Text += scrap.Prepend
			}
			itemResult.Text += output

			if scrap.CardElement != "" {
				itemResult.CardElement = scrap.CardElement
			}

			if itemResult.Name == "UID" {
				itemResult.UID = GetMD5Hash(itemResult.Text)
				result.UID = itemResult.UID
			}

			if scrap.Follow {
				itemResult.Follow = true
				itemResult.Scrap = scrap
				fmt.Println("Enable follow flag")
			} else {
				for _, subScrap := range scrap.Scraps {
					subResult := doScrap(wd, item, subScrap, testMode)
					if subResult.Follow {
						followSub = true
						fmt.Println("Subscrap has follow------------------")
					}
					if subResult.UID != "" {
						itemResult.UID = subResult.UID
						result.UID = itemResult.UID
					}
					itemResult.ScrapResults = append(itemResult.ScrapResults, subResult)
				}
			}

			result.ScrapResults = append(result.ScrapResults, itemResult)
		} else {
			fmt.Println("Empty result")
		}

		fmt.Println("End for loop")
	}

	if followSub {
		fmt.Println("Some sub need follow------------------")
		parentURL, err := wd.CurrentURL()
		if err != nil {
			panic(err)
		}
		i := 0
		for itemResultIndex, itemResult := range result.ScrapResults {
			for itemSubResultIndex, itemSubResult := range itemResult.ScrapResults {
				if itemSubResult.Follow {
					fmt.Println("one sub with follow------------------")
					if itemSubResult.Text != "" {
						followResults := followLink(wd, itemSubResult, testMode)
						result.ScrapResults[itemResultIndex].ScrapResults[itemSubResultIndex].ScrapResults = append(itemSubResult.ScrapResults, followResults...)
					} else {
						if len(itemSubResult.ScrapResults) > 0 {
							for iri, ir := range itemSubResult.ScrapResults {
								followResults := followLink(wd, ir, testMode)
								fmt.Printf("followResults : %+v\n", followResults)
								result.ScrapResults[itemResultIndex].ScrapResults[itemSubResultIndex].ScrapResults[iri].ScrapResults = append(ir.ScrapResults, followResults...)
							}
						}
					}
					i++
				}
				if testMode && i >= 1 {
					fmt.Printf("Test Mode On so only processing one child \n")
					break
				}
			}
			if testMode && i >= 1 {
				fmt.Printf("Test Mode On so only processing one child \n")
				break
			}
		}
		err = wd.Get(parentURL)
		if err != nil {
			panic(err)
		}
	}
	return result
}

func followLink(wd selenium.WebDriver, itemResult ScrapResult, testMode bool) []ScrapResult {
	fmt.Printf("follow : %s \n", itemResult.Text)
	time.Sleep(time.Duration(1+rand.Intn(6)) * time.Second)
	err := wd.Get(itemResult.Text)
	if err != nil {
		panic(err)
	}
	time.Sleep(10 * time.Second)
	var followResult []ScrapResult
	for _, subScrap := range itemResult.Scrap.Scraps {
		subResult := doScrap(wd, nil, subScrap, testMode)
		followResult = append(followResult, subResult)
	}
	return followResult
}

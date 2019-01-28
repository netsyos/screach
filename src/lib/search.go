package lib

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/tebeka/selenium"
)

type Search struct {
	Name                     string  `json:"name"`
	Template                 string  `json:"template"`
	StartURL                 string  `json:"startURL"`
	Slack                    string  `json:"slack"`
	ResultBoardShortLink     string  `json:"resultBoardShortLink"`
	IncomingResultColumnName string  `json:"incomingResultColumnName"`
	Paginator                Scrap   `json:"paginator"`
	Scraps                   []Scrap `json:"scraps"`
}

type Scrap struct {
	Name        string  `json:"name"`
	FindBy      string  `json:"findBy"`
	Location    bool    `json:"location"`
	Selector    string  `json:"selector"`
	CardElement string  `json:"cardElement"`
	Prepend     string  `json:"prepend"`
	DomField    string  `json:"domField"`
	Do          string  `json:"do"`
	Follow      bool    `json:"follow"`
	Scraps      []Scrap `json:"scraps"`
}

func (search Search) DoSearch(wd selenium.WebDriver, config Config) {
	fmt.Printf("Do Search\n")
	var searchTemplate Search
	if search.Template != "" {
		for _, template := range config.Templates {
			if template.Template == search.Template {
				searchTemplate = template
				break
			}
		}
		if search.Name != "" {
			searchTemplate.Name = search.Name
		}
		if search.Slack != "" {
			searchTemplate.Slack = search.Slack
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

	fmt.Printf("Screarch : %s\n", search.Name)
	if search.Slack != "" {
		var jsonStr = []byte(fmt.Sprintf(`{"text":"Screach : %s"}`, search.Name))
		req, err := http.NewRequest("POST", search.Slack, bytes.NewBuffer(jsonStr))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
	}

	trelloBoard := Trello{
		appKey:                   config.Trello.AppKey,
		token:                    config.Trello.Token,
		resultBoardShortLink:     search.ResultBoardShortLink,
		incomingResultColumnName: search.IncomingResultColumnName,
	}
	trelloBoard.Init()
	trelloBoard.getCards()
	mapAPI := MapAPI{
		Key:          config.GoogleAPI.Key,
		Country:      config.GoogleAPI.Country,
		Destinations: config.GoogleAPI.Destinations,
	}
	mapAPI.Init()

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
				_, found := trelloBoard.findCardByUID(r.UID)
				if !found {

					fmt.Printf("No card with UID %s card will be added\n", r.UID)
					r.exportResultToTrelloList(&trelloBoard, &mapAPI)
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
			if scrap.Location {
				itemResult.Location = true
				fmt.Println("Enable location flag")
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

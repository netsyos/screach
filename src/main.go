package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/draganshadow/trello"
	"github.com/gorilla/mux"
	"github.com/tebeka/selenium"
)

type Config struct {
	SeleniumHost             string `json:"seleniumHost"`
	SeleniumPort             string `json:"seleniumPort"`
	AppKey                   string `json:"appKey"`
	Token                    string `json:"token"`
	Member                   string `json:"member"`
	URL                      string `json:"url"`
	CSSSelector              string `json:"cssSelector"`
	ResultBoardShortLink     string `json:"resultBoardShortLink"`
	IncomingResultColumnName string `json:"incomingResultColumnName"`
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
	config := readConfig()
	r := mux.NewRouter()
	r.HandleFunc("/status/{service}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		service := vars["service"]

		fmt.Fprintf(w, "You've requested the status : %s\n", service)
	})

	// http.ListenAndServe(":80", r)

	client := trello.NewClient(config.AppKey, config.Token)

	resultBoard, err := client.GetBoard(config.ResultBoardShortLink, trello.Defaults())
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
		if rblist.Name == config.IncomingResultColumnName {
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
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://%s:%s/wd/hub", config.SeleniumHost, config.SeleniumPort))
	if err != nil {
		panic(err)
	}
	defer wd.Quit()

	// Navigate to the simple playground interface.
	if err := wd.Get(config.URL); err != nil {
		panic(err)
	}

	// Wait for the program to finish running and get the output.
	items, err := wd.FindElements(selenium.ByCSSSelector, config.CSSSelector)
	if err != nil {
		panic(err)
	}

	var itemOutput string
	var output string
	for {
		for _, item := range items {
			itemOutput, err = item.Text()
			card := trello.Card{
				Name: itemOutput,
				Desc: "Description",
			}
			err := incomingResultList.AddCard(&card, trello.Defaults())
			if err != nil {
				//Handle
			}
			output += itemOutput
		}
		if err != nil {
			panic(err)
		}
		if output != "Waiting for remote server..." {
			break
		}
		time.Sleep(time.Millisecond * 100)
	}

	fmt.Printf("%s", strings.Replace(output, "\n\n", "\n", -1))
}

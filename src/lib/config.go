package lib

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
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

func (c *Config) ReadConfig() {
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
	json.Unmarshal(byteValue, &c)
}

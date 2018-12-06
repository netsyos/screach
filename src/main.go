package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"screach/lib"
	"time"

	"github.com/gorilla/mux"
	"github.com/tebeka/selenium"
)

func main() {
	var err error
	var config lib.Config
	config.ReadConfig()
	fmt.Printf("config : %+v\n", config)

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

	caps := selenium.Capabilities{
		"browserName": "firefox",
		// "moz:firefoxOptions": f,
	}
	var wd selenium.WebDriver
	for {
		seleniumURL := fmt.Sprintf("http://%s:%s/wd/hub", config.SeleniumHost, config.SeleniumPort)
		wd, err = selenium.NewRemote(caps, seleniumURL)

		if err != nil {
			fmt.Println("Wait Selenium to be ready on " + seleniumURL)
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
		s.DoSearch(wd, config)
	}
}

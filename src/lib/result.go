package lib

import (
	"fmt"
	"log"
	"strings"

	"github.com/draganshadow/trello"
	"github.com/kr/pretty"
	yaml "gopkg.in/yaml.v2"
)

type ScrapResult struct {
	UID          string
	CardElement  string
	Name         string
	Text         string
	Location     bool
	Follow       bool
	Scrap        Scrap
	ScrapResults []ScrapResult
}

type ScrapResultData struct {
	Name        string
	Description string
	Attachments []ScrapResultAttachmentData
	UID         string
	Data        map[string]interface{}
}

type ScrapResultAttachmentData struct {
	Name string
	URL  string
}

func (result *ScrapResult) exportResultToTrelloList(trelloBoard *Trello, mapAPI *MapAPI) {
	// fmt.Printf("exportResultToTrelloList\n")
	card, attachments := result.resultToCard(mapAPI)
	err := trelloBoard.incomingResultList.AddCard(&card, trello.Defaults())
	for _, a := range attachments {
		fmt.Printf("Add attach\n")
		err := card.AttachURL(a.Name, a.URL)
		if err != nil {
			//Handle
		}
	}
	if err != nil {
		//Handle
	}
}

func (result *ScrapResult) resultToData(mapAPI *MapAPI) ScrapResultData {
	rd := ScrapResultData{
		Name: "",
		UID:  "",
		Data: make(map[string]interface{}),
	}
	if result.Text != "" {
		if result.CardElement == "name" {
			rd.Name = result.Text
		}
		if result.CardElement == "description" {
			rd.Description = strings.Title(result.Name) + " : " + result.Text
		}
		if result.CardElement == "attachment" {
			attachment := ScrapResultAttachmentData{
				Name: result.Name,
				URL:  result.Text,
			}
			rd.Attachments = append(rd.Attachments, attachment)
		}
		if result.Name == "UID" {
			result.Text = GetMD5Hash(result.Text)
			rd.UID = result.Text
		}

		if result.Name != "" {
			if result.CardElement != "attachment" {
				rd.Data[strings.Title(result.Name)] = result.Text
			}
		}
		if result.Location {

			geocodeResult := mapAPI.GeoCode(result.Text)
			if len(geocodeResult) > 0 {
				rd.Data["Lat"] = geocodeResult[0].Geometry.Location.Lat
				rd.Data["Lng"] = geocodeResult[0].Geometry.Location.Lng

				if len(mapAPI.Destinations) > 0 {
					times := make(map[string]string)
					for _, dest := range mapAPI.Destinations {
						tt := mapAPI.GetTravelTime(dest, result.Text)
						if len(tt.Rows) > 0 {
							d := tt.Rows[0].Elements[0].Duration
							sec := int(d.Seconds())
							hours := sec / 3600
							min := (sec % 3600) / 60
							times[dest] = fmt.Sprintf("%dh%02d", hours, min)
						}
					}
					rd.Data["TravelTime"] = times
					pretty.Println(times)
				}
			}
		}
	}

	for _, r := range result.ScrapResults {
		subData := r.resultToData(mapAPI)
		if subData.Name != "" {
			if rd.Name != "" {
				rd.Name += " / "
			}
			rd.Name += subData.Name
		}

		if subData.Description != "" {
			if rd.Description != "" {
				rd.Description += "\n\n\n"
			}
			rd.Description += subData.Description
		}

		if len(subData.Attachments) > 0 {
			rd.Attachments = append(rd.Attachments, subData.Attachments...)
		}

		if subData.UID != "" {
			rd.UID = subData.UID
		}

		for k, v := range subData.Data {
			rd.Data[k] = v
		}
	}

	return rd
}

func (srd *ScrapResultData) getYAML() string {

	y, err := yaml.Marshal(&srd.Data)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	return string(y)
}

func (result *ScrapResult) resultToCard(mapAPI *MapAPI) (trello.Card, []trello.Attachment) {

	card := trello.Card{
		Name: "",
		Desc: "",
	}

	rd := result.resultToData(mapAPI)

	card.Name = rd.Name
	card.Desc = rd.Description + "\n\n-----------------------------\n\n" + rd.getYAML()

	var attachments []trello.Attachment

	for _, a := range rd.Attachments {
		attachment := trello.Attachment{
			Name: a.Name,
			URL:  a.URL,
		}
		attachments = append(attachments, attachment)
	}

	return card, attachments
}

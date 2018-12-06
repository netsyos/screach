package lib

import (
	"fmt"
	"log"
	"strings"

	"github.com/draganshadow/trello"
	yaml "gopkg.in/yaml.v2"
)

type ScrapResult struct {
	UID          string
	CardElement  string
	Name         string
	Text         string
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

func (result *ScrapResult) exportResultToTrelloList(trelloBoard *Trello) {
	// fmt.Printf("exportResultToTrelloList\n")
	card := result.resultToCard()
	err := trelloBoard.incomingResultList.AddCard(&card, trello.Defaults())
	for i := len(card.Attachments) - 1; i >= 0; i-- {
		fmt.Printf("Add attach\n")
		a := card.Attachments[i]
		card.AttachURL(a.Name, a.URL)
	}

	if err != nil {
		//Handle
	}
}

func (result *ScrapResult) resultToData() ScrapResultData {
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

	}

	for _, r := range result.ScrapResults {
		subData := r.resultToData()
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

func (result *ScrapResult) resultToCard() trello.Card {

	card := trello.Card{
		Name: "",
		Desc: "",
	}

	rd := result.resultToData()

	card.Name = rd.Name
	card.Desc = rd.Description + "\n\n-----------------------------\n\n" + rd.getYAML()

	for _, a := range rd.Attachments {
		attachment := trello.Attachment{
			Name: a.Name,
			URL:  a.URL,
		}
		card.Attachments = append(card.Attachments, &attachment)
	}

	return card
}

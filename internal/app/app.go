package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/k3a/html2text"
	"github.com/mmcdole/gofeed"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"rss-parser/internal/config"
	"strings"
	"syscall"
	"time"
)

const grant_type_refresh = "refresh_token"
const link_field_id = "381047"
const place_field_id = "384301"
const upworkPlace = "Upwork"
const access_token_file = "access_token"
const refresh_token_file = "refresh_token"

var access_token = ""
var refresh_token = ""
var lastParsedTime time.Time

type CreateLeadsResponse struct {
	Embedded struct {
		Leads []struct {
			Id int `json:"id"`
		} `json:"leads"`
	} `json:"_embedded"`
}
type RefreshTokenResponse struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Item struct {
	*gofeed.Item
	Filters map[string]bool
}

func Run() {

	cfg, err := config.Init()
	if err != nil {
		log.Panicln(err)
		return
	}

	go func() {
		for {
			parse(*cfg, cfg.ParseLinkUpwork)
			time.Sleep(time.Minute * 5)
		}
	}()

	//
	// Graceful Shutdown
	//
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	log.Println("Started.")
	<-quit
	log.Println("Shutdown...")
	return

}

func parse(cfg config.Config, parseLink string) {
	access_token = readFile(access_token_file)
	refresh_token = readFile(refresh_token_file)

	newToken, err := refreshToken(cfg)
	if err != nil {
		log.Panicln(err)
	}
	access_token = newToken.AccessToken
	refresh_token = newToken.RefreshToken

	var searchTags = regexp.MustCompile(cfg.FiltersStr)

	fp := gofeed.NewParser()
	feed, _ := fp.ParseURL(parseLink)
	var results []Item

	for _, item := range feed.Items {
		if lastParsedTime.Before(*item.PublishedParsed) {
			if searchTags.MatchString(strings.ToLower(item.Description)) {
				newItem := new(Item)
				newItem.Item = item
				newItem.Filters = make(map[string]bool)

				matches := searchTags.FindAllString(strings.ToLower(item.Description), -1)

				for _, v := range matches {
					if len(newItem.Filters) < 5 {
						newItem.Filters[v] = true
					}
				}
				results = append(results, *newItem)
			}
		}
	}

	lastParsedTime = *feed.Items[0].PublishedParsed

	for _, item := range results {
		createdLeads, err := createLead(item, cfg)
		if err != nil {
			log.Fatal(err)
		}
		for _, l := range createdLeads.Embedded.Leads {
			createNote(l.Id, item, cfg)

		}
	}
}

func createLead(item Item, cfg config.Config) (result CreateLeadsResponse, err error) {
	httpClient := &http.Client{}
	tagsStr := KeysString(item.Filters)
	postBody := []byte(fmt.Sprintf(`[{
"name":"%v",
"_embedded": {
            "tags": [
                %v
            ]
        },
"custom_fields_values": [
{"field_id": %v,"values": [{"value": "%v"}]},
{"field_id": %v,"values": [{"value": "%v"}]}
] }]`, item.Title, tagsStr, link_field_id, item.Link, place_field_id, upworkPlace))
	responseBody := bytes.NewBuffer(postBody)
	//log.Println(string(postBody))
	req, _ := http.NewRequest("POST", cfg.AmoCrmEndPoint+"/api/v4/leads", responseBody)
	req.Header.Set("Authorization", "Bearer "+access_token)
	response, err := httpClient.Do(req)
	if err != nil {
		log.Fatalf("An Error Occured %v", err)
	}
	defer response.Body.Close()
	//body, _ := ioutil.ReadAll(response.Body)
	//fmt.Println("response Body:", string(body))

	err = json.NewDecoder(response.Body).Decode(&result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// Отправить примечание к сделке
func createNote(leadId int, item Item, cfg config.Config) {
	httpClient := &http.Client{}
	println("createNote")
	log.Println("qqqqq")
	postBody := []byte(fmt.Sprintf(`[{"note_type": "common","params": {"text":  "%v"}}]`, html2text.HTML2Text(item.Description)))
	responseBody := bytes.NewBuffer(postBody)
	req, _ := http.NewRequest("POST", fmt.Sprintf(cfg.AmoCrmEndPoint+"/api/v4/leads/%v/notes", leadId), responseBody)
	req.Header.Set("Authorization", "Bearer "+access_token)
	response, err := httpClient.Do(req)
	if err != nil {
		log.Fatalf("An Error Occured %v", err)
	}
	defer response.Body.Close()
	body, _ := ioutil.ReadAll(response.Body)
	fmt.Println("response Body:", string(body))
}

func refreshToken(cfg config.Config) (result RefreshTokenResponse, err error) {
	httpClient := &http.Client{}
	var refreshJson RefreshJson
	refreshJson.RefreshToken = refresh_token
	refreshJson.ClientSecret = cfg.ClientSecret
	refreshJson.ClientId = cfg.ClientId
	refreshJson.RedirectUri = cfg.RedirectUri
	refreshJson.GrantType = grant_type_refresh
	postBody, err := json.Marshal(refreshJson)
	if err != nil {
		fmt.Println(err)
		return result, err
	}
	log.Println(string(postBody))
	req, _ := http.NewRequest("POST", cfg.AmoCrmEndPoint+"/oauth2/access_token", bytes.NewBuffer(postBody))
	req.Header.Set("Content-Type", "application/json")

	response, err := httpClient.Do(req)
	if err != nil {
		log.Fatalf("An Error Occured %v", err)
	}
	defer response.Body.Close()
	//body, _ := ioutil.ReadAll(response.Body)
	//fmt.Println("response Body:", string(body))
	err = json.NewDecoder(response.Body).Decode(&result)
	if err != nil {
		return result, err
	}

	err = writeFile(access_token_file, result.AccessToken)
	if err != nil {
		return RefreshTokenResponse{}, err

	}
	err = writeFile(refresh_token_file, result.RefreshToken)
	if err != nil {
		return RefreshTokenResponse{}, err
	}
	return result, nil
}

type RefreshJson struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	GrantType    string `json:"grant_type"`
	RefreshToken string `json:"refresh_token"`
	RedirectUri  string `json:"redirect_uri"`
}

func KeysString(m map[string]bool) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return "{\"name\": \"" + strings.Join(keys, "\"}, {\"name\": \"") + "\"}"
}

func readFile(filepath string) string {
	content, err := os.ReadFile(filepath)
	if err != nil {
		log.Fatal(err)
	}
	return string(content)
}

func writeFile(filepath string, value string) error {
	err := ioutil.WriteFile(filepath, []byte(value), 0777)
	if err != nil {
		return err
	}
	return nil
}

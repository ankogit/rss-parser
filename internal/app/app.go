package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/jomei/notionapi"
	"github.com/k3a/html2text"
	"github.com/mmcdole/gofeed"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"rss-parser/internal/config"
	"strconv"
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
const close_status = 143
const dbId = "ba14a6981f5f4efeb3e1cf274a38b1e1"
const upwork = "Upwork"
const fl = "fl.ru"
const newStatus = "Новый"
const rgxCountry = "country<\\/b>: (.*)\\n"
const rgxBudget = "budget<\\/b>: (.*)\\n"
const rgxSkills = "skills<\\/b>:(.*)\\n"
const rgxHourlyRange = "hourly range<\\/b>: (.*)\\n"

const rgxBudgetFL = "\\(бюджет: (.*)\\  &#8381;\\)"

var access_token = ""
var refresh_token = ""
var lastParsedTime time.Time
var lastParsedTimeFL time.Time

type CreateLeadsResponse struct {
	Embedded struct {
		Leads []struct {
			Id        int   `json:"id"`
			CreatedAt int64 `json:"created_at"`
		} `json:"leads"`
	} `json:"_embedded"`
}

type UpdateLeadsRequest struct {
	Id       int `json:"id"`
	StatusId int `json:"status_id"`
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
	Country string
	Budget  int
	Hourly  string
	Type    string
	Source  string
	Skills  []string
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
			parseFL(*cfg, "https://www.fl.ru/rss/all.xml?category=2")
			time.Sleep(time.Minute * 5)
		}
	}()

	go func() {
		for {
			removeOldLeads(*cfg)
			time.Sleep(time.Hour * 6)
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

func auth(cfg config.Config) {
	access_token = readFile(access_token_file)
	refresh_token = readFile(refresh_token_file)

	newToken, err := refreshToken(cfg)
	if err != nil {
		log.Fatal(err)
	}
	access_token = newToken.AccessToken
	refresh_token = newToken.RefreshToken
}

func parseFL(cfg config.Config, parseLink string) {
	notionClient := notionapi.NewClient(notionapi.Token(cfg.NotionSecret))
	var budgetTags = regexp.MustCompile(rgxBudgetFL)
	var results []Item

	fp := gofeed.NewParser()
	feed, _ := fp.ParseURL(parseLink)

	for _, item := range feed.Items {
		if lastParsedTimeFL.Before(*item.PublishedParsed) {
			newItem := new(Item)
			newItem.Item = item

			matchesBudget := budgetTags.FindStringSubmatch(strings.ToLower(item.Title))
			//log.Println("matchesBudget ", matchesBudget)
			if isset(matchesBudget, 1) && matchesBudget[1] != "" {
				newItem.Budget, _ = strconv.Atoi(strings.Replace(matchesBudget[1], "$", "", -1))
				newItem.Type = "budget"
			} else {
				newItem.Type = "non fixed"
			}
			newItem.Skills = item.Categories
			newItem.Source = fl
			results = append(results, *newItem)
			//log.Println("Find one fl")
			//log.Println(results)

		}

	}
	//
	lastParsedTimeFL = *feed.Items[0].PublishedParsed

	for _, item := range results {
		createNotionPage(notionClient, item)
		createdLeads, err := createLead(item, cfg)
		log.Println("Lead was created")

		if err != nil {
			log.Println(err)
		}
		for _, l := range createdLeads.Embedded.Leads {
			createNote(l.Id, item, cfg)
		}
	}
}

func parse(cfg config.Config, parseLink string) {
	auth(cfg)

	notionClient := notionapi.NewClient(notionapi.Token(cfg.NotionSecret))

	var searchTags = regexp.MustCompile(cfg.FiltersStr)
	var countryTags = regexp.MustCompile(rgxCountry)
	var budgetTags = regexp.MustCompile(rgxBudget)
	var skillsTags = regexp.MustCompile(rgxSkills)
	var hourlyTags = regexp.MustCompile(rgxHourlyRange)
	var excludedSearchTags = regexp.MustCompile(cfg.ExcludedFiltersStr)

	fp := gofeed.NewParser()
	feed, _ := fp.ParseURL(parseLink)
	var results []Item

	for _, item := range feed.Items {
		if lastParsedTime.Before(*item.PublishedParsed) {
			if excludedSearchTags.MatchString(strings.ToLower(item.Description)) {
				continue
			}
			if searchTags.MatchString(strings.ToLower(item.Description)) {
				newItem := new(Item)
				newItem.Item = item
				newItem.Filters = make(map[string]bool)

				//newItem

				matches := searchTags.FindAllString(strings.ToLower(item.Description), -1)

				for _, v := range matches {
					if len(newItem.Filters) < 5 {
						newItem.Filters[v] = true
					}
				}

				matchesCountry := countryTags.FindStringSubmatch(strings.ToLower(item.Description))
				if isset(matchesCountry, 1) && matchesCountry[1] != "" {
					newItem.Country = matchesCountry[1]
				}
				matchesBudget := budgetTags.FindStringSubmatch(strings.ToLower(item.Description))
				if isset(matchesBudget, 1) && matchesBudget[1] != "" {
					newItem.Budget, _ = strconv.Atoi(strings.Replace(matchesBudget[1], "$", "", -1))
					newItem.Type = "budget"

				}
				matchesSkills := skillsTags.FindStringSubmatch(strings.ToLower(item.Description))
				if isset(matchesSkills, 1) && matchesSkills[1] != "" {
					skillStr := strings.Trim(strings.Replace(matchesSkills[1], "$", "", -1), " ")
					newItem.Skills = strings.Split(skillStr, ",     ")
				}
				matchesHourly := hourlyTags.FindStringSubmatch(strings.ToLower(item.Description))

				if isset(matchesHourly, 1) && matchesHourly[1] != "" {
					newItem.Hourly = matchesHourly[1]
					newItem.Type = "hourly"
				}
				log.Println("matchesBudget:", newItem, matchesBudget, strings.ToLower(item.Description))
				//for _, v := range matchesCountry {
				//	//if len(newItem.Filters) < 5 {
				//	//	newItem.Filters[v] = true
				//	//}
				//}

				newItem.Source = upwork
				results = append(results, *newItem)
				log.Println("Find one")
			}
		}
	}

	lastParsedTime = *feed.Items[0].PublishedParsed

	for _, item := range results {
		createNotionPage(notionClient, item)
		createdLeads, err := createLead(item, cfg)
		log.Println("Lead was created")

		if err != nil {
			log.Println(err)
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
	req, _ := http.NewRequest("POST", cfg.AmoCrmEndPoint+"/api/v4/leads", responseBody)
	req.Header.Set("Authorization", "Bearer "+access_token)
	response, err := httpClient.Do(req)
	if err != nil {
		log.Fatalf("An Error Occured %v", err)
	}
	defer response.Body.Close()
	//body, _ := ioutil.ReadAll(response.Body)
	//fmt.Println("response Body:", string(body))
	//log.Println(access_token)
	err = json.NewDecoder(response.Body).Decode(&result)
	if err != nil {
		return result, err
	}
	log.Println(result)
	return result, nil
}

// Отправить примечание к сделке
func createNote(leadId int, item Item, cfg config.Config) {
	httpClient := &http.Client{}
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

func removeOldLeads(cfg config.Config) error {
	log.Println("Start removing")
	auth(cfg)
	httpClient := &http.Client{}
	req, _ := http.NewRequest("GET", cfg.AmoCrmEndPoint+"/api/v4/leads?page=1&limit=100&filter[statuses][0][status_id]=54816778&filter[statuses][0][pipeline_id]=6406018", nil)
	req.Header.Set("Authorization", "Bearer "+access_token)
	response, err := httpClient.Do(req)
	if err != nil {
		log.Fatalf("An Error Occured %v", err)
	}
	defer response.Body.Close()
	//body, _ := ioutil.ReadAll(response.Body)
	//log.Println("response Body:", string(body))
	//log.Println(access_token)

	var result CreateLeadsResponse
	err = json.NewDecoder(response.Body).Decode(&result)
	if err != nil {
		return err
	}

	var request []UpdateLeadsRequest

	for _, lead := range result.Embedded.Leads {
		// Calling Unix method
		if lead.CreatedAt <= time.Now().AddDate(0, 0, -2).Unix() {
			request = append(request, UpdateLeadsRequest{
				Id:       lead.Id,
				StatusId: close_status,
			})
		}
	}

	postBody, err := json.Marshal(request)
	responseBody := bytes.NewBuffer(postBody)
	req, _ = http.NewRequest("PATCH", cfg.AmoCrmEndPoint+"/api/v4/leads", responseBody)
	req.Header.Set("Authorization", "Bearer "+access_token)
	response, _ = httpClient.Do(req)
	//if err != nil {
	//	//log.Fatalf("An Error Occured %v", err)
	//}
	defer response.Body.Close()
	//body, _ := ioutil.ReadAll(response.Body)
	//fmt.Println("response Body:", string(body))
	log.Println("End removing")

	return nil
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

	return strings.TrimSuffix(string(content), "\n")
}

func writeFile(filepath string, value string) error {
	err := ioutil.WriteFile(filepath, []byte(value), 0777)
	if err != nil {
		return err
	}
	return nil
}

func createNotionPage(client *notionapi.Client, item Item) {

	var techOptionsFilters []notionapi.Option
	for filter, _ := range item.Filters {
		techOptionsFilters = append(techOptionsFilters, notionapi.Option{
			Name: filter,
		})
	}
	var techOptions []notionapi.Option
	if len(item.Skills) == 0 {
		techOptions = techOptionsFilters
	}
	for _, filter := range item.Skills {
		techOptions = append(techOptions, notionapi.Option{
			Name: filter,
		})
	}

	country := "Russia"
	if len(item.Country) > 0 {
		country = item.Country
	}

	pageRequest := &notionapi.PageCreateRequest{Parent: notionapi.Parent{
		Type:       notionapi.ParentTypeDatabaseID,
		DatabaseID: dbId,
	},
		Properties: notionapi.Properties{
			"Название": notionapi.TitleProperty{
				Title: []notionapi.RichText{
					{Text: &notionapi.Text{Content: item.Title}},
				},
			},
		}}

	if len(item.Source) > 0 {
		pageRequest.Properties["Ресурс"] = notionapi.SelectProperty{
			Select: notionapi.Option{
				Name: item.Source,
			},
		}
	}

	pageRequest.Properties["Статус"] = notionapi.SelectProperty{
		Select: notionapi.Option{
			Name: newStatus,
		},
	}

	if len(techOptions) > 0 {
		pageRequest.Properties["Технологии"] = notionapi.MultiSelectProperty{
			MultiSelect: techOptions,
		}
	}
	if len(techOptionsFilters) > 0 {
		pageRequest.Properties["Фильтры"] = notionapi.MultiSelectProperty{
			MultiSelect: techOptionsFilters,
		}
	}
	if len(item.Link) > 0 {
		pageRequest.Properties["URL"] = notionapi.URLProperty{
			URL: item.Link,
		}
	}
	if len(item.Type) > 0 {
		pageRequest.Properties["Тип"] = notionapi.RichTextProperty{
			RichText: []notionapi.RichText{
				{
					Text: &notionapi.Text{
						Content: item.Type,
					},
					//PlainText: item.Type,
				},
			},
		}
	}
	if item.Budget != 0 {
		pageRequest.Properties["Бюджет"] = notionapi.NumberProperty{
			Number: float64(item.Budget),
		}
	}
	if len(country) > 0 {
		pageRequest.Properties["Страна"] = notionapi.SelectProperty{
			Select: notionapi.Option{
				Name: strings.Title(country),
			},
		}
	}
	if len(item.Hourly) > 0 {
		pageRequest.Properties["Hourly Range"] = notionapi.RichTextProperty{
			RichText: []notionapi.RichText{
				{
					Text: &notionapi.Text{
						Content: item.Hourly,
					},
				},
			},
		}
	}

	if len(item.Description) > 0 {
		pageRequest.Children = []notionapi.Block{
			notionapi.Heading1Block{
				BasicBlock: notionapi.BasicBlock{
					Object: notionapi.ObjectTypeBlock,
					Type:   notionapi.BlockTypeHeading1,
				},
				Heading1: notionapi.Heading{
					RichText: []notionapi.RichText{
						{
							Type: notionapi.ObjectTypeText,
							Text: &notionapi.Text{Content: "Background info"},
						},
					},
				},
			},
			notionapi.ParagraphBlock{
				BasicBlock: notionapi.BasicBlock{
					Object: notionapi.ObjectTypeBlock,
					Type:   notionapi.BlockTypeParagraph,
				},
				Paragraph: notionapi.Paragraph{
					RichText: []notionapi.RichText{
						{
							Text: &notionapi.Text{
								Content: html2text.HTML2Text(item.Description),
							},
						},
					},
					Children: nil,
				},
			},
		}
	}

	page, err := client.Page.Create(context.Background(), pageRequest)
	//notionapi.PageCreateRequest{
	//	Parent: notionapi.Parent{
	//		Type:       notionapi.ParentTypeDatabaseID,
	//		DatabaseID: dbId,
	//	},
	//	Properties: notionapi.Properties{
	//		"Название": notionapi.TitleProperty{
	//			Title: []notionapi.RichText{
	//				{Text: &notionapi.Text{Content: item.Title}},
	//			},
	//		},
	//		"Ресурс": notionapi.SelectProperty{
	//			Select: notionapi.Option{
	//				Name: item.Source,
	//			},
	//		},
	//		"Статус": notionapi.SelectProperty{
	//			Select: notionapi.Option{
	//				Name: newStatus,
	//			},
	//		},
	//		"Технологии": notionapi.MultiSelectProperty{
	//			MultiSelect: techOptions,
	//		},
	//		"Фильтры": notionapi.MultiSelectProperty{
	//			MultiSelect: techOptionsFilters,
	//		},
	//		"URL": notionapi.URLProperty{
	//			URL: item.Link,
	//		},
	//		"Тип": notionapi.RichTextProperty{
	//			RichText: []notionapi.RichText{
	//				{
	//					Text: &notionapi.Text{
	//						Content: item.Type,
	//					},
	//					//PlainText: item.Type,
	//				},
	//			},
	//		},
	//		"Бюджет": notionapi.NumberProperty{
	//			Number: float64(item.Budget),
	//		},
	//		"Страна": notionapi.SelectProperty{
	//			Select: notionapi.Option{
	//				Name: strings.Title(country),
	//			},
	//		},
	//
	//		"Hourly Range": notionapi.RichTextProperty{
	//			RichText: []notionapi.RichText{
	//				{
	//					Text: &notionapi.Text{
	//						Content: item.Hourly,
	//					},
	//				},
	//			},
	//		},
	//	},
	//	Children: []notionapi.Block{
	//		notionapi.Heading1Block{
	//			BasicBlock: notionapi.BasicBlock{
	//				Object: notionapi.ObjectTypeBlock,
	//				Type:   notionapi.BlockTypeHeading1,
	//			},
	//			Heading1: notionapi.Heading{
	//				RichText: []notionapi.RichText{
	//					{
	//						Type: notionapi.ObjectTypeText,
	//						Text: &notionapi.Text{Content: "Background info"},
	//					},
	//				},
	//			},
	//		},
	//		notionapi.ParagraphBlock{
	//			BasicBlock: notionapi.BasicBlock{
	//				Object: notionapi.ObjectTypeBlock,
	//				Type:   notionapi.BlockTypeParagraph,
	//			},
	//			Paragraph: notionapi.Paragraph{
	//				RichText: []notionapi.RichText{
	//					{
	//						Text: &notionapi.Text{
	//							Content: html2text.HTML2Text(item.Description),
	//						},
	//					},
	//				},
	//				Children: nil,
	//			},
	//		},
	//	},
	//})

	if err != nil {
		log.Println(err)
	}
	log.Println(page)
}

func isset(arr []string, index int) bool {
	return (len(arr) > index)
}

package app

import (
	"fmt"
	"github.com/jomei/notionapi"
	"github.com/mehanizm/airtable"
	"github.com/mmcdole/gofeed"
	"io/ioutil"
	"log"
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
const flDevWeb = "fl.ru / Разработка сайтов"
const flDesign = "fl.ru / Дизайн сайтов"
const flDesignWeb = "fl.ru / Дизайн сайтов (Интерфейсы)"
const flDesignApp = "fl.ru / Дизайн сайтов (Дизайн интерфейсов приложений)"
const flMobileApp = "fl.ru / Мобильные приложения"
const flDevCRM = "fl.ru / Разработка CRM и ERP"
const flArc = "fl.ru / Проектирование"
const flIntApp = "fl.ru / Интерактивные приложения"
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
var lastParsedTimeFLDesign time.Time
var lastParsedTimeFLDesignWeb time.Time
var lastParsedTimeFLDesignApp time.Time
var lastParsedTimeFLMobileApp time.Time
var lastParsedTimeFLDevCRM time.Time
var lastParsedTimeFLArc time.Time
var lastParsedTimeFLIntApp time.Time

type ParsingClients struct {
	NotionClient   *notionapi.Client
	AirTableClient *airtable.Client
	AirTableTable  *airtable.Table
}

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
	Filters    map[string]bool
	Country    string
	Budget     int
	HourlyFrom float64
	HourlyTo   float64
	Hourly     string
	Type       string
	Source     string
	Skills     []string
}

func Run() {

	cfg, err := config.Init()
	if err != nil {
		log.Panicln(err)
		return
	}
	notionClient := notionapi.NewClient(notionapi.Token(cfg.NotionSecret))
	airTableClient := airtable.NewClient(cfg.AirTableSecret)

	//airTable := airTableClient.GetTable(cfg.AirTableDatabase, cfg.AirTableTable)
	airTableUpwork := airTableClient.GetTable(cfg.AirTableDatabase, cfg.AirTableTableUpwork)
	airTableFL := airTableClient.GetTable(cfg.AirTableDatabase, cfg.AirTableTableFL)

	//parsingClients := ParsingClients{
	//	NotionClient:   notionClient,
	//	AirTableClient: airTableClient,
	//	AirTableTable:  airTable,
	//}
	parsingClientsUpwork := ParsingClients{
		NotionClient:   notionClient,
		AirTableClient: airTableClient,
		AirTableTable:  airTableUpwork,
	}
	parsingClientsFL := ParsingClients{
		NotionClient:   notionClient,
		AirTableClient: airTableClient,
		AirTableTable:  airTableFL,
	}
	go func() {
		for {
			parseUpwork(*cfg, parsingClientsUpwork, cfg.ParseLinkUpwork)

			//parseFL(*cfg, parsingClients, "https://www.fl.ru/rss/all.xml?category=2", flDevWeb, &lastParsedTimeFL)

			//Дизайн сайтов
			parseFL(*cfg, parsingClientsFL, "https://www.fl.ru/rss/all.xml?subcategory=172&category=3", flDesign, &lastParsedTimeFLDesign)

			//Дизайн сайтов (Интерфейсы)
			parseFL(*cfg, parsingClientsFL, "https://www.fl.ru/rss/all.xml?subcategory=35&category=3", flDesignWeb, &lastParsedTimeFLDesignWeb)

			//Дизайн сайтов (Дизайн интерфейсов приложений)
			parseFL(*cfg, parsingClientsFL, "https://www.fl.ru/rss/all.xml?subcategory=239&category=3", flDesignApp, &lastParsedTimeFLDesignApp)

			//Мобильные приложения
			//parseFL(*cfg, parsingClients, "https://www.fl.ru/rss/all.xml?category=23", flMobileApp, &lastParsedTimeFLMobileApp)

			//Разработка CRM и ERP
			//parseFL(*cfg, parsingClients, "https://www.fl.ru/rss/all.xml?subcategory=222&category=5", flDevCRM, &lastParsedTimeFLDevCRM)

			//Проектирование
			//parseFL(*cfg, parsingClients, "https://www.fl.ru/rss/all.xml?subcategory=133&category=5", flArc, &lastParsedTimeFLArc)

			//Интерактивные приложения
			//parseFL(*cfg, parsingClients, "https://www.fl.ru/rss/all.xml?subcategory=223&category=5", flIntApp, &lastParsedTimeFLIntApp)

			time.Sleep(time.Minute * 5)
		}
	}()

	//go func() {
	//	for {
	//		removeOldLeads(*cfg)
	//		time.Sleep(time.Hour * 6)
	//	}
	//}()

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

func parseFL(cfg config.Config, parseClients ParsingClients, parseLink string, source string, timer *time.Time) {
	var budgetTags = regexp.MustCompile(rgxBudgetFL)
	var results []Item

	fp := gofeed.NewParser()
	feed, _ := fp.ParseURL(parseLink)

	for _, item := range feed.Items {
		if timer.Before(*item.PublishedParsed) {
			newItem := new(Item)
			newItem.Item = item

			matchesBudget := budgetTags.FindStringSubmatch(strings.ToLower(item.Title))
			if isset(matchesBudget, 1) && matchesBudget[1] != "" {
				newItem.Budget, _ = strconv.Atoi(strings.Replace(matchesBudget[1], "$", "", -1))
				newItem.Type = "budget"
			} else {
				newItem.Type = "non fixed"
			}
			newItem.Skills = item.Categories
			newItem.Source = source
			results = append(results, *newItem)
		}

	}
	timer = feed.Items[0].PublishedParsed

	createRecords(cfg, parseClients, results)
}

func parseUpwork(cfg config.Config, parseClients ParsingClients, parseLink string) {
	var searchTags = regexp.MustCompile(fmt.Sprintf("\\b(%v)\\b", cfg.FiltersStr))
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
					var err error
					newItem.Budget, err = strconv.Atoi(strings.Replace(matchesBudget[1], "$", "", -1))
					if err != nil {
						log.Println(err)
					}
					newItem.Type = "budget"
				} else {
					newItem.Type = "non fixed"
				}
				matchesSkills := skillsTags.FindStringSubmatch(strings.ToLower(item.Description))
				if isset(matchesSkills, 1) && matchesSkills[1] != "" {
					skillStr := strings.Trim(strings.Replace(matchesSkills[1], "$", "", -1), " ")
					newItem.Skills = strings.Split(skillStr, ",     ")
				}
				matchesHourly := hourlyTags.FindStringSubmatch(strings.ToLower(item.Description))

				if isset(matchesHourly, 1) && matchesHourly[1] != "" {
					newItem.Hourly = matchesHourly[1]
					rates := strings.Split(newItem.Hourly, "-")
					if len(rates) == 2 {
						newItem.HourlyFrom, _ = strconv.ParseFloat(strings.Replace(rates[0], "$", "", -1), 32)
						newItem.HourlyTo, _ = strconv.ParseFloat(strings.Replace(rates[1], "$", "", -1), 32)
					}

					newItem.Type = "hourly"
				}
				newItem.Source = upwork
				results = append(results, *newItem)
				log.Println("Find one")
			}
		}
	}

	lastParsedTime = *feed.Items[0].PublishedParsed

	createRecords(cfg, parseClients, results)
}

func createRecords(cfg config.Config, clients ParsingClients, results []Item) {
	if clients.AirTableClient != nil {
		createAirtableRecords(clients.AirTableTable, results)
	}

	if clients.NotionClient != nil {
		for _, item := range results {
			createNotionPage(clients.NotionClient, item)
		}
	}

}

func KeysString(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
func KeysStringAmoCRM(m map[string]bool) string {
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

func isset(arr []string, index int) bool {
	return (len(arr) > index)
}

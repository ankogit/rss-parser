package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/k3a/html2text"
	"io/ioutil"
	"log"
	"net/http"
	"rss-parser/internal/config"
	"time"
)

func createLeadAmoCRM(item Item, cfg config.Config) (result CreateLeadsResponse, err error) {
	httpClient := &http.Client{}
	tagsStr := KeysStringAmoCRM(item.Filters)
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

func authAmoCRM(cfg config.Config) {
	access_token = readFile(access_token_file)
	refresh_token = readFile(refresh_token_file)

	newToken, err := refreshTokenAmoCRM(cfg)
	if err != nil {
		log.Fatal(err)
	}
	access_token = newToken.AccessToken
	refresh_token = newToken.RefreshToken
}

// Отправить примечание к сделке
func createNoteAmoCRM(leadId int, item Item, cfg config.Config) {
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

func removeOldLeadsAmoCRM(cfg config.Config) error {
	log.Println("Start removing")
	authAmoCRM(cfg)
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

func refreshTokenAmoCRM(cfg config.Config) (result RefreshTokenResponse, err error) {
	httpClient := &http.Client{}
	var refreshJson RefreshJsonAmoCRM
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

type RefreshJsonAmoCRM struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	GrantType    string `json:"grant_type"`
	RefreshToken string `json:"refresh_token"`
	RedirectUri  string `json:"redirect_uri"`
}

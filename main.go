package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

type SecretData struct {
	AuthToken string `json:"authToken"`
}

type InputData struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type ResponseData struct {
	Data struct {
		Listing_id string `json:"listing_id"`
		Provider   string `json:"provider"`
		Start_date string `json:"start_date"`
		End_date   string `json:"end_date"`
		Days       []struct {
			Date    string `json:"date"`
			Day     string `json:"day"`
			MinStay int    `json:"min_stay"`
			Status  struct {
				Reason    string `json:"reason"`
				Available bool   `json:"available"`
			}
		} `json:"days"`
	}
}

// type ResponseData2 struct {
// 	Data Data
// }
// type Data struct {
// 	Listing_id string `json:"listing_id"`
// 	Provider   string `json:"provider"`
// 	Start_date string `json:"start_date"`
// 	End_date   string `json:"end_date"`
// 	Days       Days   `json:"days"`
// }

// type Days struct {
// 	Date    string `json:"date"`
// 	Day     string `json:"day"`
// 	MinStay int    `json:"min_stay"`
// 	Status  Status `json:"status"`
// }

// type Status struct {
// 	Reason    string `json:"reason"`
// 	Available bool   `json:"available"`
// }

var (
	secretName           string = "smartbnb_token_1"
	region               string = "us-east-1"
	versionStage         string = "AWSCURRENT"
	availProperties      []string
	availPropertiesNicks []string
)

func main() {
	lambda.Start(HandleRequest)

}

func HandleRequest(ctx context.Context, input InputData) ([]string, error) {
	lc, _ := lambdacontext.FromContext(ctx)
	fmt.Println("Context: ", lc.ClientContext)

	inputJSON, _ := json.Marshal(input)
	fmt.Println("Input: ", string(inputJSON))

	start_date := input.StartDate
	end_date := input.EndDate

	dwellPropertyIds := []string{"119966", "529490", "625432", "164360", "119676"}
	lucilePropertyIds := []string{"155944", "156010", "156008", "155942"}
	sharpePropertyIds := []string{"623998", "624000", "628594", "633472", "650394", "650416"}
	franklinPropertyIds := []string{"164362"}
	complexes := [][]string{dwellPropertyIds, lucilePropertyIds, sharpePropertyIds, franklinPropertyIds}

	token := get_token()

	c := make(chan string)
	var wg sync.WaitGroup

	for complex := range complexes {
		for propertyId := range complexes[complex] {
			property := complexes[complex][propertyId]
			wg.Add(1)
			go checkAvailability(property, start_date, end_date, token, c, &wg)
		}
	}

	go func() {
		wg.Wait()
		close(c)
	}()

	for p := range c {
		availProperties = append(availProperties, p)
	}

	if len(availProperties) != 0 {
		fmt.Println("availProperties", availProperties)
	} else {
		fmt.Println("No available properties")
	}
	availPropertiesNicks, _ := getListingByProperty(availProperties)
	return (availPropertiesNicks), nil
}

// Check if a property is available
func checkAvailability(propertyId string, start_date string, end_date string, token string, c chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Println("Checking availability for property: ", propertyId)

	base_url := "https://api.hospitable.com/calendar/"
	url := base_url + propertyId + "?start_date=" + start_date + "&end_date=" + end_date
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		fmt.Println(err)
		return
	}

	req.Header.Add("Authorization", token)
	req.Header.Add("Content-Type", "application/vnd.smartbnb.20210721+json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	responsedata := ResponseData{}
	json.Unmarshal(body, &responsedata)

	daysLength := len(responsedata.Data.Days)
	daysCounter := 0

	for day := range responsedata.Data.Days {
		daysCounter += 1
		if !responsedata.Data.Days[day].Status.Available {
			break
		} else if daysCounter == daysLength {
			c <- propertyId
		}
	}
}

// grab a token from AWS Secrets Manager
func get_token() string {
	svc := secretsmanager.New(
		session.New(),
		aws.NewConfig().WithRegion(region),
	)

	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretName),
		VersionStage: aws.String(versionStage),
	}

	result, err := svc.GetSecretValue(input)
	if err != nil {
		panic(err.Error())
	}

	var secretString string
	if result.SecretString != nil {
		secretString = *result.SecretString
	}

	var secretData SecretData
	err = json.Unmarshal([]byte(secretString), &secretData)
	if err != nil {
		panic(err.Error())
	}

	return secretData.AuthToken
}

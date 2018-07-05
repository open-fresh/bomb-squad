package prom

import (
	"io/ioutil"
	"log"
	"net/http"
)

type InstantQuery struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string          `json:"resultType"`
		Result     []InstantResult `json:"result"`
	} `json:"data"`
}

type InstantResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}
}

type Series struct {
	Status string              `json:"status"`
	Data   []map[string]string `json:"data"`
}

// Fetch queries prometheus over http at a given endpoint and returns the body
func Fetch(endpt string, client *http.Client) ([]byte, error) {
	req, _ := http.NewRequest("GET", endpt, nil)

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error in response from p8s client", err)
		return []byte{}, err
	}

	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	return body, nil
}

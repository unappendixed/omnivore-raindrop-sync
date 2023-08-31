package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

var output chan string
var omni_response chan []byte

func main() {
	fmt.Printf("Hello, world!\n")
	srv := make_server()

	godotenv.Load(".env")

	output = make(chan string)
	omni_response = make(chan []byte)

	go func() { srv.ListenAndServe() }()

	go parse_omnivore_response(<-omni_response)

	srv.Close()

	fmt.Printf("%v\n", <-output)

}

func make_server() *http.Server {
	server := http.Server{
		Addr:    ":8080",
		Handler: nil,
	}

	http.HandleFunc("/", handle)

	return &server
}

func handle(w http.ResponseWriter, req *http.Request) {

	bytes, err := io.ReadAll(req.Body)

	if err != nil {
		panic(err)
	}

	omni_response <- bytes

	w.WriteHeader(http.StatusOK)
	w.Header().Add("content-type", "text/plain")
	w.Write([]byte("Good job!"))
}

func parse_omnivore_response(omni_body []byte) {

	type OmniBody struct {
		Page struct {
			Url string
		}
	}

	fmt.Println(string(omni_body[:]))
	data := OmniBody{}
	err := json.Unmarshal(omni_body, &data)

	if err != nil {
		panic(err)
	}

	original_url := data.Page.Url

	valid_ch := make(chan bool)
	check_raindrop_exists(valid_ch, original_url)

	raindrop_exists := <-valid_ch

	if !raindrop_exists {
		//create_raindrop()
	}
}

func create_raindrop(result chan string, target_url string) {

}

func check_raindrop_exists(valid chan bool, target_url string) {

	raindrop_check_url := "https://api.raindrop.io/rest/v1/import/url/exists"

	body_string := fmt.Sprintf(`{
		"urls": [
			"%v"
		]
	}`, target_url)

	token := os.Getenv("RAINDROP_TOKEN")

	req_body := bytes.NewReader([]byte(body_string))

	req, err := http.NewRequest(http.MethodPost, raindrop_check_url, req_body)
	if err != nil {
		panic(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Add("Content-Type", "application/json")

	response_channel := make(chan []byte)

	go func() {
		resp, err := http.DefaultClient.Do(req)

		if err != nil {
			panic(err)
		}

		res_body, err := io.ReadAll(resp.Body)

		if err != nil {
			panic(err)
		}

		response_channel <- res_body
	}()

	response_body_bytes := <-response_channel
	response_body := struct{ result bool }{}

	if err := json.Unmarshal(response_body_bytes, &response_body); err != nil {
		panic(err)
	}

	valid <- response_body.result

}

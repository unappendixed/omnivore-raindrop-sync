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

type NewRaindropBookmark struct {
	Url   string
	Title string
}

func main() {

	srv := http.Server{
		Addr:    ":8080",
		Handler: nil,
	}

	omni_response := make(chan []byte)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { handle(w, r, omni_response) })

	godotenv.Load(".env")

	//TODO add graceful shutdown logic
	go func() { srv.ListenAndServe() }()

	fmt.Printf("Server started successfully\nWaiting for requests...")

	for {

		data := parse_omnivore_response(<-omni_response)

		fmt.Fprintf(os.Stderr, "Received %q\n", data.Url)

		valid_ch := make(chan bool)

		go check_raindrop_exists(valid_ch, data.Url)

		result := make(chan string)

		valid := <-valid_ch

		if valid {
			go create_raindrop(result, &data)
		} else {
			go func() { result <- "Bookmark already in Raindrop.io bookmarks" }()
		}

		fmt.Fprintf(os.Stderr, "%v\n", <-result)
	}

}

// handles the request and sends the request body through ch channel
func handle(w http.ResponseWriter, req *http.Request, ch chan []byte) {

	bytes, err := io.ReadAll(req.Body)

	if err != nil {
		panic(err)
	}

	ch <- bytes

	w.WriteHeader(http.StatusOK)
}

func parse_omnivore_response(omni_body []byte) NewRaindropBookmark {

	fmt.Println(string(omni_body[:]))

	// init anonymous struct to unmarshal json body
	data := struct {
		Page struct {
			Url   string
			Title string
		}
	}{}

	err := json.Unmarshal(omni_body, &data)

	if err != nil {
		panic(err)
	}

	return NewRaindropBookmark{
		Url:   data.Page.Url,
		Title: data.Page.Title,
	}
}

func create_raindrop(result chan string, bookmark *NewRaindropBookmark) {

	endpoint := "https://api.raindrop.io/rest/v1/raindrop"

	data := fmt.Sprintf(`
	{
		"link": "%v",
		"title": "%v",
		"pleaseParse": {}
	}`, bookmark.Url, bookmark.Title)

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader([]byte(data)))

	if err != nil {
		panic(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", os.Getenv("RAINDROP_TOKEN")))
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		panic(err)
	} else if res.StatusCode != http.StatusOK {
		result <- fmt.Sprintf("Unexpected response from Raindrop.io api: %q", res.Status)
	}

	result <- "Successfully created Raindrop.io bookmark."
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

	response_body := struct{ Result bool }{}

	if err := json.Unmarshal(response_body_bytes, &response_body); err != nil {
		panic(err)
	}

	valid <- !response_body.Result

}

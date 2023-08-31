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

	godotenv.Load(".env")

    if os.Getenv("RAINDROP_TOKEN") == "" {
        panic("RAINDROP_TOKEN env variable is empty!")
    }

	srv := http.Server{
		Addr:    ":8080",
		Handler: nil,
	}

	omniResponse := make(chan []byte)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { handle(w, r, omniResponse) })

	//TODO add graceful shutdown logic
	go func() { srv.ListenAndServe() }()

	fmt.Printf("Server started successfully\nWaiting for requests...\n")

	for {

		data := parseOmnivoreResponse(<-omniResponse)

		fmt.Fprintf(os.Stderr, "Received %q\n", data.Url)

        valid := checkRaindropExists(data.Url)

        var result string

		if valid {
            result = createRaindrop(&data)
		} else {
			result = "Bookmark already in Raindrop.io bookmarks"
		}

		fmt.Fprintf(os.Stderr, "%v\n", result)
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

func parseOmnivoreResponse(omniBody []byte) NewRaindropBookmark {

	// init anonymous struct to unmarshal json body
	data := struct {
		Page struct {
			Url   string
			Title string
		}
	}{}

	err := json.Unmarshal(omniBody, &data)

	if err != nil {
		panic(err)
	}

	return NewRaindropBookmark{
		Url:   data.Page.Url,
		Title: data.Page.Title,
	}
}

func createRaindrop(bookmark *NewRaindropBookmark) string {

	endpoint := "https://api.raindrop.io/rest/v1/raindrop"

	body := fmt.Sprintf(`
	{
		"link": "%v",
		"title": "%v",
		"pleaseParse": {}
	}`, bookmark.Url, bookmark.Title)

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader([]byte(body)))

	if err != nil {
		panic(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", os.Getenv("RAINDROP_TOKEN")))
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		panic(err)
	} else if res.StatusCode != http.StatusOK {
		return fmt.Sprintf("Unexpected response from Raindrop.io api: %q", res.Status)
	}

	return "Successfully created Raindrop.io bookmark."
}

func checkRaindropExists(targetUrl string) bool {

	endpoint := "https://api.raindrop.io/rest/v1/import/url/exists"

    body := struct {
        Urls []string `json:"urls"`
    }{Urls: []string{targetUrl}}

    bodyString, err := json.Marshal(body)

    if err != nil {
        panic(err)
    }

	token := os.Getenv("RAINDROP_TOKEN")

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyString))
	if err != nil {
		panic(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		panic(err)
	} else if resp.StatusCode >= 400 {
        panic(resp.Status)
    }

	rawResponse, err := io.ReadAll(resp.Body)

	if err != nil {
		panic(err)
	}

	responseData := struct{ Result bool }{}

	if err := json.Unmarshal(rawResponse, &responseData); err != nil {
		panic(err)
	}

	//raindrop api returns false if there *isn't* a duplicate, so we negate this
	return !responseData.Result

}

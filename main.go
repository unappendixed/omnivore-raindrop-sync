package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type NewRaindropBookmark struct {
	Url    string
	Title  string
	UserId string
}

func stderrHelper(template string, v ...any) {
    time := time.Now().Local()
    timeS := fmt.Sprintf("%v/%v/%v %v:%v:%v", time.Year(), int(time.Month()), time.Day(), time.Hour(), time.Minute(), time.Second())
    s := fmt.Sprintf(template, v...)
    fmt.Fprintf(os.Stderr, "%v\t%v", timeS, s)
}

func main() {

	godotenv.Load(".env")

	if os.Getenv("RAINDROP_TOKEN") == "" {
		panic("RAINDROP_TOKEN env variable is empty!")
	}

    if os.Getenv("OMNIVORE_USERID") == "" {
        panic("OMNIVORE_USERID env variable is empty!")
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

		data, err := parseOmnivoreResponse(<-omniResponse)

        if err != nil {
            stderrHelper("Parse Error: %v\n", err.Error())
            continue
        }

		if data.UserId != os.Getenv("OMNIVORE_USERID") {
			stderrHelper("Rejecting request from invalid userId %q\n", data.UserId)
			continue
		}

		stderrHelper("Received %q\n", data.Url)

		valid, err := checkRaindropExists(data.Url)

        if err != nil {
            stderrHelper("Raindrop Check Response: %v\n", err.Error())
            continue
        }

		if valid {
            err := createRaindrop(&data)
            if err != nil {
                stderrHelper("Create Raindrop Response: %v\n", err.Error())
                continue
            }
		} else {
			stderrHelper("Bookmark already in Raindrop.io bookmarks\n")
            continue
		}
        stderrHelper("Successfully created Raindrop.io bookmark\n")
	}

}

// handles the request and sends the request body through ch channel
func handle(w http.ResponseWriter, req *http.Request, ch chan []byte) {

	bytes, err := io.ReadAll(req.Body)

	if err != nil {
		fmt.Fprint(os.Stderr, err)
		return
	}

	ch <- bytes

	w.WriteHeader(http.StatusOK)
}

func parseOmnivoreResponse(omniBody []byte) (NewRaindropBookmark, error) {

	// init anonymous struct to unmarshal json body
	data := struct {
		UserId string `json:"userID"`
		Page   struct {
			Url   string
			Title string
		}
	}{}

	err := json.Unmarshal(omniBody, &data)

	if err != nil {
        return NewRaindropBookmark{}, err
	}

	return NewRaindropBookmark{
        Url: data.Page.Url, 
        Title: data.Page.Title, 
        UserId: data.UserId,
    }, nil
}

func createRaindrop(bookmark *NewRaindropBookmark) error {

	endpoint := "https://api.raindrop.io/rest/v1/raindrop"

	body := fmt.Sprintf(`
	{
		"link": "%v",
		"title": "%v",
		"pleaseParse": {}
	}`, bookmark.Url, bookmark.Title)

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader([]byte(body)))

	if err != nil {
		return err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", os.Getenv("RAINDROP_TOKEN")))
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)

	if err != nil {
        return err
	} else if res.StatusCode != http.StatusOK {
		return fmt.Errorf("Unexpected response from Raindrop.io api: %q", res.Status)
	}

	return nil
}

func checkRaindropExists(targetUrl string) (bool, error) {

	endpoint := "https://api.raindrop.io/rest/v1/import/url/exists"

	body := struct {
		Urls []string `json:"urls"`
	}{Urls: []string{targetUrl}}

	bodyString, err := json.Marshal(body)

	if err != nil {
		return false, err
	}

	token := os.Getenv("RAINDROP_TOKEN")

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyString))
	if err != nil {
        return false, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return false, err
	} else if resp.StatusCode >= 400 {
        return false, fmt.Errorf(resp.Status)
	}

	rawResponse, err := io.ReadAll(resp.Body)

	if err != nil {
		return false, err
	}

	responseData := struct{ Result bool }{}

	if err := json.Unmarshal(rawResponse, &responseData); err != nil {
        return false, err
	}

	//raindrop api returns false if there *isn't* a duplicate, so we negate this
	return !responseData.Result, nil

}

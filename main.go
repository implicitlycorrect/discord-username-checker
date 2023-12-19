package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/a-h/templ"
)

const (
	serverAddress  = "127.0.0.1:8080"
	timeoutSeconds = 10
)

func main() {
	index := Index()

	http.Handle("/", templ.Handler(index))
	http.Handle("/check-username-available", http.HandlerFunc(handleCheckUsernameAvailability))

	fmt.Printf("Listening on %s\n", serverAddress)
	if err := http.ListenAndServe(serverAddress, nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func handleCheckUsernameAvailability(writer http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		handleError(writer, fmt.Sprintf("Error parsing form data. Error: %v", err), http.StatusInternalServerError)
		return
	}

	username := request.Form.Get("username")

	taken, err := isUsernameTaken(username)

	if err != nil {
		handleError(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	renderCheckUsernameAvailability(writer, username, taken)
}

func isUsernameTaken(username string) (bool, error) {
	requestBody, err := json.Marshal(map[string]string{"username": username})
	if err != nil {
		return false, fmt.Errorf("error encoding request body: %v", err)
	}

	httpClient := &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second}
	req, err := http.NewRequest("POST", "https://discord.com/api/v9/unique-username/username-attempt-unauthed", bytes.NewBuffer(requestBody))
	if err != nil {
		return false, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	response, err := httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("error making request: %v", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return false, fmt.Errorf("error reading response body: %v", err)
	}

	trimmedBody := strings.TrimSpace(string(body))
	switch trimmedBody {
	case `{"taken":true}`:
		return true, nil
	case `{"taken":false}`:
		return false, nil
	default:
		return false, fmt.Errorf("invalid response: %s", trimmedBody)
	}
}

func handleError(writer http.ResponseWriter, errorMessage string, statusCode int) {
	writer.WriteHeader(statusCode)
	errorComponent := ErrorPage(errorMessage)
	if err := errorComponent.Render(context.Background(), writer); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
}

func renderCheckUsernameAvailability(writer http.ResponseWriter, username string, taken bool) {
	writer.WriteHeader(http.StatusOK)
	availabilityComponent := CheckUsernameAvailability(username, taken)
	if err := availabilityComponent.Render(context.Background(), writer); err != nil {
		handleError(writer, err.Error(), http.StatusInternalServerError)
	}
}
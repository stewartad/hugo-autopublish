package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-github/v43/github"
)

var (
	GITHUB_SECRET  string
	LOCAL_REPO_DIR string
	SERVER_DIR     string
	GITHUB_TOKEN   string
)

// setGlobalVars sets global configuration settings from environment variables set by processEnvFile
func setGlobalVars() {
	GITHUB_SECRET = os.Getenv("GITHUB_SECRET")
	LOCAL_REPO_DIR = os.Getenv("LOCAL_REPO_DIR")
	SERVER_DIR = os.Getenv("SERVER_DIR")
	GITHUB_TOKEN = os.Getenv("GITHUB_TOKEN")
}

// processEnvFile reads configuration settings from a file named "env"
func processEnvFile() {
	confFile, err := os.Open("env")
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(confFile)
	for scanner.Scan() {
		k, v, found := strings.Cut(scanner.Text(), "=")
		if !found {
			log.Fatal("Invalid config file")
		}
		os.Setenv(k, v)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func publish() {
	hugoCmd := exec.Command("hugo", "-d", SERVER_DIR)
	hugoCmd.Dir = LOCAL_REPO_DIR
	err := hugoCmd.Run()
	if err != nil {
		log.Printf("Error running Hugo %s\n", err)
	}
	log.Printf("Site published\n")
}

func pullRepo() {
	// check local git directory exists
	_, err := os.Stat(LOCAL_REPO_DIR)
	if err != nil {
		if os.IsExist(err) {
			log.Printf("Local repo %s does not exist\n", LOCAL_REPO_DIR)
		} else {
			log.Printf("unknown error %s\n", err)
		}
	}

	// pull updates to local repo
	pullCmd := exec.Command("git", "pull")
	pullCmd.Dir = LOCAL_REPO_DIR
	err = pullCmd.Run()
	if err != nil {
		log.Printf("Could not pull repo to %s: %s\n", LOCAL_REPO_DIR, err)
	}
}

// processPushEvent updates the local repository copy and publishes the site
func processPushEvent(e *github.PushEvent) {
	pullRepo()
	publish()
	// TODO: automated webmention sending
	// TODO: automated syndication backfeeds
}

// handleGithubWebhook listens for updates from GitHub and calls the appropriate function
func handleGithubWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, []byte(GITHUB_SECRET))
	if err != nil {
		log.Printf("Error reading request body: err=%s\n", err)
		return
	}
	defer r.Body.Close()

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		log.Printf("Could not parse webhook: err=%s\n", err)
		return
	}

	switch e := event.(type) {
	case *github.PushEvent:
		processPushEvent(e)
	default:
		log.Printf("Event type %s. Ignoring.\n", github.WebHookType(r))
		return
	}
}

func main() {
	processEnvFile()
	setGlobalVars()
	log.Println("Server Started")
	http.HandleFunc("/github", handleGithubWebhook)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	GitHubAPIURL  string `yaml:"github_api_url"`
	RepoOwner     string `yaml:"repo_owner"`
	RepoName      string `yaml:"repo_name"`
	GitHubToken   string `yaml:"github_token"`
	ServerURL     string `yaml:"server_url"`
	LastCommitSHA string `yaml:"commit_sha"`
}

var config Config

func main() {
	loadConfig()

	newCommitSHA, err := getLatestMainCommitSHA()
	if err != nil {
		log.Fatalf("Failed to get latest commit SHA: %v", err)
	}

	if newCommitSHA == config.LastCommitSHA {
		log.Println("No updates in 'main' branch.")
		return
	}

	log.Printf("Detected new commit: %s. Initiating merge...", newCommitSHA)

	config.LastCommitSHA = newCommitSHA
	saveConfig()

	if err := triggerMergeOnServer(); err != nil {
		log.Fatalf("Failed to initiate merge: %v", err)
	}

	log.Println("Merge request successfully initiated.")
}

func loadConfig() {
	file, err := os.Open("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config.yaml: %v", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		log.Fatalf("Failed to parse config.yaml: %v", err)
	}
}

func saveConfig() {
	file, err := os.Create("config.yaml")
	if err != nil {
		log.Fatalf("Failed to save config.yaml: %v", err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	if err := encoder.Encode(&config); err != nil {
		log.Fatalf("Failed to encode config.yaml: %v", err)
	}
}

func getLatestMainCommitSHA() (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/branches/main", config.GitHubAPIURL, config.RepoOwner, config.RepoName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "token "+config.GitHubToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API responded with status: %s", resp.Status)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	commitData, ok := result["commit"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected JSON structure: 'commit' key not found")
	}

	sha, ok := commitData["sha"].(string)
	if !ok {
		return "", fmt.Errorf("unexpected JSON structure: 'sha' key not found")
	}

	return sha, nil
}

func triggerMergeOnServer() error {
	resp, err := http.Get(fmt.Sprintf("%s/merge-main", config.ServerURL))
	if err != nil {
		return fmt.Errorf("failed to trigger merge on server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server responded with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

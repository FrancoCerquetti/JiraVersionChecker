package main

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/fatih/color"
)

type Credentials struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

type IssueUser struct {
	Name string `json:"displayName"`
}

type IssueStatus struct {
	Description string `json:"description"`
}

type IssueFields struct {
	Summary  string      `json:"summary"`
	Reporter IssueUser   `json:"reporter"`
	Assignee IssueUser   `json:"assignee"`
	Status   IssueStatus `json:"status"`
	Evidence string      `json:"customfield_17840"`
}

type Issue struct {
	Key         string `json:"key"`
	IssueFields `json:"fields"`
}

type IssueList struct {
	Issues []Issue `json:"issues"`
}

func main() {
	project, version := obtainCLIArgs()

	// Obtain credentials
	contentBytes, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Error opening credentials file: %v", err)
	}
	var credentials Credentials
	if err := json.Unmarshal(contentBytes, &credentials); err != nil {
		log.Fatalf("Error parsing credentials: %v", err)
	}

	// Create and make the request
	request := createHttpRequest(credentials, project, version)
	var client *http.Client = &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		log.Fatalf("Error performing request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("Oops something went wrong with the request. Status: %v", resp.Status)
	}

	// Read and parse the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	var issueList IssueList
	if err := json.Unmarshal(body, &issueList); err != nil {
		log.Fatalf("Error parsing response body: %v", err)
	}

	printIssueList(issueList)
}

func createHttpRequest(credentials Credentials, projectArg string, versionArg string) *http.Request {
	credentialsConcat := fmt.Sprintf("%s:%s", credentials.User, credentials.Password)
	credentialsEncoded := b64.StdEncoding.EncodeToString([]byte(credentialsConcat))
	auth := fmt.Sprintf("Basic %s", credentialsEncoded)

	project := fmt.Sprintf("project=%s", projectArg)
	fixVersion := fmt.Sprintf("fixVersion=%s", versionArg)
	url := "https://jira.despegar.com/rest/api/2/search/?jql=" + project + "%20AND%20" + fixVersion + "&fields=summary,status,fixVersions,reporter,assignee,customfield_17840"

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}

	request.Header.Add("Authorization", auth)

	return request
}

func printIssueList(issueList IssueList) {
	if len(issueList.Issues) == 0 {
		fmt.Println("No issues found!")
		return
	}

	for _, issue := range issueList.Issues {
		var jiraKey = color.YellowString(issue.Key)
		fmt.Printf("- [%s] %s\n", jiraKey, issue.Summary)
		fmt.Printf("\t- %s\n", formatJiraURL(issue.Key))
		fmt.Printf("\t- Informador: %s\n", issue.Reporter.Name)
		fmt.Printf("\t- Responsable: %s\n", issue.Assignee.Name)
		fmt.Printf("\t- Status: %s\n", issue.Status.Description)
		fmt.Printf("\t- Evidencia: %v\n", issueEvidenceToSimpleString(issue.Evidence))
	}
}

func issueEvidenceToSimpleString(evidence string) string {
	if evidence != "" {
		return color.GreenString("Completa")
	}
	return color.RedString("Incompleta")
}

func formatJiraURL(issueKey string) string {
	return fmt.Sprintf("https://jira.despegar.com/browse/%s", issueKey)
}

func obtainCLIArgs() (string, string) {
	if len(os.Args) < 3 {
		log.Fatal("Wrong number of arguments. Usage: ./version_checker <project> <version>")
	}

	project := os.Args[1]
	version := os.Args[2]

	return project, version
}

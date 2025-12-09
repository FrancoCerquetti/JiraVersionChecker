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

type IssueUserDTO struct {
	Name string `json:"displayName"`
}

type IssueStatusDTO struct {
	Description string `json:"description"`
}

type IssueFieldsDTO struct {
	Summary  string         `json:"summary"`
	Reporter IssueUserDTO   `json:"reporter"`
	Assignee IssueUserDTO   `json:"assignee"`
	Status   IssueStatusDTO `json:"status"`
	Evidence string         `json:"customfield_17840"`
}

type IssueDTO struct {
	Key            string `json:"key"`
	IssueFieldsDTO `json:"fields"`
}

type IssueDTOList struct {
	Issues []IssueDTO `json:"issues"`
}

type Issue struct {
	Key               string
	Summary           string
	URL               string
	Reporter          string
	Assignee          string
	Status            string
	EvidenceCompleted bool
}

type IssueList struct {
	Issues []Issue
}

func (issue Issue) evidenceString() string {
	if issue.EvidenceCompleted {
		return color.GreenString("Completa")
	}
	return color.RedString("Incompleta")
}

func (issueList IssueList) printIssues() {
	if len(issueList.Issues) == 0 {
		fmt.Println("No issues found!")
		return
	}

	for _, issue := range issueList.Issues {
		var jiraKey = color.YellowString(issue.Key)
		fmt.Printf("- [%s] %s\n", jiraKey, issue.Summary)
		fmt.Printf("\t- %s\n", issue.URL)
		fmt.Printf("\t- Informador: %s\n", issue.Reporter)
		fmt.Printf("\t- Responsable: %s\n", issue.Assignee)
		fmt.Printf("\t- Status: %s\n", issue.Status)
		fmt.Printf("\t- Evidencia: %v\n", issue.evidenceString())
	}
}

func (issueList IssueList) printIssuesWithoutEvidence() {
	if len(issueList.Issues) == 0 {
		return
	}

	var issuesWithoutEvidence []Issue

	for _, issue := range issueList.Issues {
		if !issue.EvidenceCompleted {
			issuesWithoutEvidence = append(issuesWithoutEvidence, issue)
		}
	}

	if len(issuesWithoutEvidence) > 0 {
		fmt.Println("\n- Jiras sin evidencia: ")
		for _, issue := range issuesWithoutEvidence {
			fmt.Printf("\t- %s: %s\n", issue.Key, issue.URL)
		}
	}
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

	var issueDtoList IssueDTOList
	if err := json.Unmarshal(body, &issueDtoList); err != nil {
		log.Fatalf("Error parsing response body: %v", err)
	}

	issueList := transformIssueDtoList(issueDtoList)
	issueList.printIssues()
	issueList.printIssuesWithoutEvidence()
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

func transformIssueDtoList(issueDtoList IssueDTOList) IssueList {
	var issues []Issue
	for _, dto := range issueDtoList.Issues {
		issues = append(issues, issueFromDTO(dto))
	}

	return IssueList{Issues: issues}
}

func issueFromDTO(dto IssueDTO) Issue {
	return Issue{
		dto.Key,
		dto.Summary,
		formatJiraURL(dto.Key),
		dto.Reporter.Name,
		dto.Assignee.Name,
		dto.Status.Description,
		dto.Evidence != "",
	}
}

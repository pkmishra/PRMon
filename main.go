package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	url2 "net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"

	"golang.org/x/oauth2"

	"github.com/google/go-github/github"
)

type Input struct {
	SlackWebHookUrl string `json:"slack_web_hook_url"`
	SlackChannel    string `json:"channel"`
	GitAccessToken  string `json:"access_token"`
	GitRepoQuery    string `json:"git_repo_query"`
	GitUser         string `json:"git_user"`
	BaseUrl         string `json:"base_url"`
}

func Handler(input Input) {
	client, _ := buildGitClient(input)

	searchOpt := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 50},
	}

	opt := &github.PullRequestListOptions{
		State: "open", ListOptions: github.ListOptions{PerPage: 20},
	}

	repos, _, err := client.Search.Repositories(context.Background(), fmt.Sprintf("%s user:%s archived:false", input.GitRepoQuery, input.GitUser), searchOpt)

	log.Println("found repos", len(repos.Repositories))
	if err != nil || len(repos.Repositories) == 0 {
		printAndExit("Couldn't fetch repo from git", err)

	}
	ps := []*github.PullRequest{}

	for _, repo := range repos.Repositories {
		//log.Printf("Traversing repo %v for pull request", repo.GetName())
		p, _, err := client.PullRequests.List(context.Background(), input.GitUser, repo.GetName(), opt)
		if err == nil {
			ps = append(ps, p...)
		}
	}

	n := len(ps)
	if len(ps) == 0 {
		err = SendSlackNotification(input.SlackWebHookUrl, input.SlackChannel, buildNoPullRequestMessage())
	} else {
		var msg strings.Builder
		msg.WriteString(buildMessageHeader(n))
		for _, p := range ps {
			msg.WriteString(buildSlackMessageBody(p))
		}
		err = SendSlackNotification(input.SlackWebHookUrl, input.SlackChannel, msg.String())
	}
	if err != nil {
		printAndExit("Couldn't post to slack", err)
	}
}

func buildGitClient(input Input) (*github.Client, error) {
	var baseUrl string
	var err error
	var client *github.Client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: input.GitAccessToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	if !isEnterPrise(input.BaseUrl) {
		client = github.NewClient(tc)
	} else {
		u, err := url2.Parse(input.BaseUrl)
		if err != nil {
			printAndExit("Can not pass baseurl ", err)
		}
		if strings.Contains(u.Path, "api") {
			baseUrl = input.BaseUrl
		} else {
			u.Path = path.Join(u.Path, "/api/v3/")
			baseUrl = u.String()
		}
		client, err = github.NewEnterpriseClient(baseUrl, baseUrl, tc)
		if err != nil {
			printAndExit("Couldn't create git client", err)
		}
	}
	return client, err
}

func isEnterPrise(url string) bool {
	return url != "" && !strings.Contains(url, "github.com")
}

func main() {

	lambda.Start(Handler)

	/* Uncomment to test locally and comment above line*/
	//var input Input
	//err := json.Unmarshal([]byte(os.Args[1]), &input)
	//if err != nil {
	//	log.Fatal(err)
	//	os.Exit(1)
	//}
	//if input.GitAccessToken == "" || input.SlackWebHookUrl == "" || input.SlackChannel == "" || input.GitRepoQuery == "" {
	//	printDefaultAndExit("SlackWebHookUrl, GitAccessToken, SlackChannel and GitUser all values are required")
	//}
	//Handler(os.Args[1])
}

func printAndExit(msg string, err error) {
	log.Fatal(msg, err)
	os.Exit(1)
}

type SlackRequestBody struct {
	Channel   string `json:"channel"`
	IconEmoji string `json:"icon_emoji"`
	Text      string `json:"text"`
}

func buildMessageHeader(n int) string {
	var msg strings.Builder
	msg.WriteString(">*Current pull requests statistics *\n")
	if n > 1 {
		msg.WriteString(fmt.Sprintf("There are %d open pull requests waiting for review :angrier_seal: :red_circle: \n", n))
	} else if n == 1 {
		msg.WriteString("There is just one open pull request waiting for review \n")
	}
	return msg.String()
}
func buildSlackMessageBody(r *github.PullRequest) string {
	var msg strings.Builder
	msg.WriteString("* <")
	msg.WriteString(r.GetHTMLURL())
	msg.WriteString("|")
	msg.WriteString(r.GetTitle())
	msg.WriteString("> in ")
	msg.WriteString(r.GetHead().GetRepo().GetName())
	msg.WriteString(" for ")
	msg.WriteString(fmt.Sprintf("*%.1f hours*", time.Since(r.GetCreatedAt()).Hours()))
	msg.WriteString(fmt.Sprintf(" | opened by <%v|%v>", r.GetUser().GetHTMLURL(), r.GetUser().GetLogin()))
	msg.WriteString("\n")
	return msg.String()
}

func buildNoPullRequestMessage() string {
	var msg strings.Builder
	msg.WriteString(buildMessageHeader(0))
	msg.WriteString("Hurray! There are no open pull requests. Good job team! :happyseal: :happyseal:")
	return msg.String()
}

func SendSlackNotification(webhookUrl string, channel string, msg string) error {

	slackBody, _ := json.Marshal(SlackRequestBody{Text: msg, Channel: channel})
	req, err := http.NewRequest(http.MethodPost, webhookUrl, bytes.NewBuffer(slackBody))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil || buf.String() != "ok" {
		return errors.New("Non-ok response returned from Slack")
	}
	return nil
}

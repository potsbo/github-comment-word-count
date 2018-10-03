package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

const (
	githubTokenKey = "GITHUB_ACCESS_TOKEN"
)

func main() {
	if err := countChars(); err != nil {
		fmt.Printf("failed to count chars: %v\n", err)
		os.Exit(1)
	}
}

func countChars() error {
	accessToken := os.Getenv(githubTokenKey)
	if len(accessToken) == 0 {
		return errors.Errorf("%s is missing", githubTokenKey)
	}
	client := getNewClient(accessToken)

	month := "09"
	nextMonth := "10"
	year := "2018"
	username := "potsbo"
	org := "wantedly"

	query := fmt.Sprintf("org:%s involves:%s updated:>%s-%s-01 created:<%s-%s-01", org, username, year, month, year, nextMonth)

	issues, err := getAllIssues(query, client)
	if err != nil {
		return errors.Wrap(err, "failed list all issues")
	}
	fmt.Printf("%d issue(s) found\n", len(issues))

	cnt := 0
	for i, issue := range issues {
		if i%10 == 0 {
			fmt.Printf("%d/%d\n", i+1, len(issues))
		}
		size, err := getComment(issue, client, username, year, month)
		if err != nil {
			return errors.Wrap(err, "failed to count chars on an issue")
		}
		cnt += size
	}
	fmt.Printf("%d char(s) written by %s\n", cnt, username)
	return nil
}

func getNewClient(accessToken string) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: accessToken,
	})
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	return client
}

func getAllIssues(query string, client *github.Client) ([]github.Issue, error) {
	var issues []github.Issue
	opt := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		res, resp, err := client.Search.Issues(context.Background(), query, opt)
		if err != nil {
			return nil, err
		}
		issues = append(issues, res.Issues...)
		fmt.Printf("%d/%d\n", len(issues), res.GetTotal())
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return issues, nil
}

func getComment(issue github.Issue, client *github.Client, username, year, month string) (int, error) {
	//TODO: filter by date
	cnt := 0
	if issue.GetUser().GetLogin() == username {
		cnt += len(issue.GetBody())
	}
	if issue.GetComments() == 0 {
		return cnt, nil
	}

	owner, repo, err := getOwnerRepoFromURL(issue.GetRepositoryURL())

	opt := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	var allComments []*github.IssueComment
	for {
		comments, resp, err := client.Issues.ListComments(context.Background(), owner, repo, issue.GetNumber(), nil)
		if err != nil {
			return 0, err
		}
		allComments = append(allComments, comments...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	if err != nil {
		return 0, errors.Wrap(err, "failed to list comments")
	}
	for _, comment := range allComments {
		if comment.GetUser().GetLogin() != username {
			continue
		}
		cnt += len(comment.GetBody())
	}
	return cnt, nil
}

func getOwnerRepoFromURL(str string) (string, string, error) {
	u, err := url.Parse(str)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to parse URL")
	}
	paths := strings.Split(u.Path, "/")
	owner := paths[2]
	repo := paths[3]
	return owner, repo, nil
}

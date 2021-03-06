package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
)

type repeatedFlag map[string]bool

func (i repeatedFlag) String() string {
	return "my string representation"
}

func (i repeatedFlag) Set(value string) error {
	i[value] = true
	return nil
}

var (
	source  = flag.String("source", "", "Base reference to copy.")
	message = flag.String("message", "", "Generate an empty commit in new branch with this message.")
	ignore  = make(repeatedFlag)
)

func myUsage() {
	fmt.Printf("Usage: %s [OPTION]... owner/repository branch:\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	flag.Usage = myUsage
	flag.Var(&ignore, "ignore", "Contexts to ignore.")
	flag.Parse()
	if flag.NArg() != 2 {
		flag.Usage()
	}
	ownerRepo := strings.Split(flag.Arg(0), "/")
	if len(ownerRepo) != 2 {
		log.Fatalf("Repository must be in the form owner/name.")
	}
	owner := ownerRepo[0]
	repo := ownerRepo[1]
	destination := flag.Arg(1)

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatalf("Please set GITHUB_TOKEN.")
	}
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	if *source == "" {
		rep, _, err := client.Repositories.Get(ctx, owner, repo)
		if err != nil {
			log.Fatalf("Unable to find repository, %v", err)
		}
		source = rep.DefaultBranch
	}

	baseRef, _, err := client.Git.GetRef(ctx, owner, repo, "refs/heads/"+*source)
	if err != nil {
		log.Fatalf("Unable to get base reference, %v", err)
	}
	baseObject := baseRef.Object

	if *message != "" {
		baseCommit, _, err := client.Git.GetCommit(ctx, owner, repo, *baseObject.SHA)
		if err != nil {
			log.Fatalf("Unable to get base commit, %v", err)
		}
		newCommit, _, err := client.Git.CreateCommit(ctx, owner, repo, &github.Commit{
			Tree:    baseCommit.Tree,
			Message: message,
			Parents: []*github.Commit{baseCommit},
		})
		if err != nil {
			log.Fatalf("Cannot create commit, %v", err)
		}
		baseObject = &github.GitObject{SHA: newCommit.SHA}
	}

	newRef := &github.Reference{Ref: github.String("refs/heads/" + destination), Object: baseObject}
	_, _, err = client.Git.CreateRef(ctx, owner, repo, newRef)
	if err != nil {
		log.Fatalf("Cannot create branch, %v", err)
	}

	pending := true
	for pending {
		time.Sleep(20 * time.Second)
		pending = false
		builds := make(map[string]bool)
		statuses, _, err := client.Repositories.ListStatuses(ctx, owner, repo, *baseObject.SHA, &github.ListOptions{})
		if err != nil {
			log.Fatalf("Cannot get statuses, %v", err)
		}
		for _, status := range statuses {
			if ignore[*status.Context] {
				continue
			}
			switch *status.State {
			case "pending":
				builds[*status.TargetURL] = builds[*status.TargetURL] || false
			case "success":
				builds[*status.TargetURL] = true
			default:
				log.Fatalf("%s: %s %s", *status.State, *status.Description, *status.TargetURL)
			}
		}
		checkRuns, _, err := client.Checks.ListCheckRunsForRef(ctx, owner, repo, *baseObject.SHA, &github.ListCheckRunsOptions{})
		if err != nil {
			log.Fatalf("Cannot get check runs, %v", err)
		}
		for _, checkRun := range checkRuns.CheckRuns {
			if ignore[*checkRun.Name] {
				continue
			}
			switch *checkRun.Status {
			case "queued":
				builds[*checkRun.HTMLURL] = false
			case "in_progress":
				builds[*checkRun.HTMLURL] = false
			case "completed":
				builds[*checkRun.HTMLURL] = true
			default:
				log.Fatalf("%s: %s %s", *checkRun.Status, *checkRun.Name, *checkRun.URL)
			}
		}
		for k, v := range builds {
			if !v {
				log.Printf("Pending: %s", k)
				pending = true
			}
		}
	}
}

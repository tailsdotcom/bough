package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"

	"github.com/google/go-github/v35/github"
	"golang.org/x/oauth2"
)

var (
	source = flag.String("source", "master", "Base reference to copy.")
)

func main() {
	flag.Parse()
	if flag.NArg() != 2 {
		log.Fatalf("Please supply repository and destination.")
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

	baseRef, _, err := client.Git.GetRef(ctx, owner, repo, "refs/heads/"+*source)
	if err != nil {
		log.Fatalf("Unable to get base reference, %v", err)
	}

	newRef := &github.Reference{Ref: github.String("refs/heads/" + destination), Object: &github.GitObject{SHA: baseRef.Object.SHA}}
	_, _, err = client.Git.CreateRef(ctx, owner, repo, newRef)
	if err != nil {
		log.Fatalf("Cannot create branch, %v", err)
	}

}

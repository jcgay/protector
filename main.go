package main

import (
	"flag"
	"fmt"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"os"
)

const (
	BANNER  = "protector - v%s\n"
	VERSION = "0.1.0-SNAPSHOT"
)

var (
	ghToken string
	dryrun  bool
	version bool
)

func init() {
	// parse flags
	flag.StringVar(&ghToken, "token", "", "GitHub API token")
	flag.BoolVar(&dryrun, "dry-run", false, "do not make any changes, just print out what would have been done")
	flag.BoolVar(&version, "version", false, "print version and exit")
	flag.BoolVar(&version, "v", false, "print version and exit (shorthand)")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, fmt.Sprintf(BANNER, VERSION))
		flag.PrintDefaults()
	}

	flag.Parse()

	if version {
		fmt.Printf("v%s", VERSION)
		os.Exit(0)
	}

	if ghToken == "" {
		usageAndExit("GitHub token cannot be empty.", 1)
	}
}

func main() {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghToken},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	client := github.NewClient(tc)

	repos := listRepositories(client, 1)

	for _, repo := range repos {
		if (*repo.Permissions)["admin"] == false {
			fmt.Printf("%s: you don't have admin rights to modify this repository\n", *repo.FullName)
			continue
		}

		err := protect(client, repo)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}

	os.Exit(0)
}

func protect(client *github.Client, repo *github.Repository) error {
	opt := &github.ListOptions{
		PerPage: 100,
	}

	branches, resp, err := client.Repositories.ListBranches(*repo.Owner.Login, *repo.Name, opt)

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	for _, branch := range branches {
		if *branch.Name == "master" {
			if *branch.Protection.Enabled {
				fmt.Printf("%s: %s is already protected\n", *repo.FullName, *branch.Name)
				return nil
			}

			fmt.Printf("%s: %s will be set to protected\n", *repo.FullName, *branch.Name)
			if dryrun {
				return nil
			}

			activateProtection := true
			branch.Protection.Enabled = &activateProtection
			if _, _, err := client.Repositories.EditBranch(*repo.Owner.Login, *repo.Name, *branch.Name, branch); err != nil {
				return err
			}
		}
	}

	return nil
}

func listRepositories(client *github.Client, startPage int) []*github.Repository {
	opt := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{
			Page:    startPage,
			PerPage: 20,
		},
	}

	repos, resp, err := client.Repositories.List("", opt)
	if err != nil {
		log.Println(err)
		return make([]*github.Repository, 0)
	}

	if startPage == resp.LastPage || resp.NextPage == 0 {
		return make([]*github.Repository, 0)
	}

	return append(repos, listRepositories(client, resp.NextPage)...)
}

func usageAndExit(message string, exitCode int) {
	if message != "" {
		fmt.Fprintf(os.Stderr, message)
		fmt.Fprint(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprint(os.Stderr, "\n")
	os.Exit(exitCode)
}

package main

import (
	"flag"
	"fmt"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
)

const (
	BANNER  = "protector - v%s\n"
	VERSION = "0.1.0-SNAPSHOT"
)

var (
	ghToken             string
	dryrun              bool
	version             bool
	unprotect           bool
	protectBranches     []*regexp.Regexp
	protectRepositories stringsFlag
)

type stringsFlag []string

func (s *stringsFlag) String() string {
	return fmt.Sprintf("%s", *s)
}
func (s *stringsFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func init() {
	// parse flags
	flag.StringVar(&ghToken, "token", "", "GitHub API token")
	flag.BoolVar(&dryrun, "dry-run", false, "do not make any changes, just print out what would have been done")
	flag.BoolVar(&version, "version", false, "print version and exit")
	flag.BoolVar(&version, "v", false, "print version and exit (shorthand)")
	flag.BoolVar(&unprotect, "free", false, "remove branch protection")
	flag.Var(&protectRepositories, "repos", "repositories fullname to protect (ex: jcgay/maven-color)")

	var branches stringsFlag
	flag.Var(&branches, "branches", "branches to include (as regexp)")

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

	protectBranches = make([]*regexp.Regexp, 0)
	for _, branch := range branches {
		protectBranches = append(protectBranches, regexp.MustCompile(branch))
	}

	if len(protectBranches) == 0 {
		protectBranches = append(protectBranches, regexp.MustCompile("master"))
	}
}

func main() {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghToken},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	client := github.NewClient(tc)

	var repos chan *github.Repository
	if len(protectRepositories) > 0 {
		repos = fetchRepositories(client, protectRepositories)
	} else {
		repos = listRepositories(client, 1)
	}

	var wg sync.WaitGroup
	for repo := range repos {
		wg.Add(1)
		go func(repository *github.Repository) {
			defer wg.Done()
			if (*repository.Permissions)["admin"] == false {
				fmt.Printf("%s: you don't have admin rights to modify this repository\n", *repository.FullName)
				return
			}

			err := process(client, repository)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}(repo)
	}
	wg.Wait()

	os.Exit(0)
}
func fetchRepositories(client *github.Client, repoFullNames []string) chan *github.Repository {
	result := make(chan *github.Repository)
	var wg sync.WaitGroup
	for _, repoFullName := range repoFullNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			metas := strings.SplitN(name, "/", 2)
			if repo, _, err := client.Repositories.Get(metas[0], metas[1]); err == nil {
				result <- repo
			}
		}(repoFullName)
	}

	go func() {
		wg.Wait()
		close(result)
	}()

	return result
}

func process(client *github.Client, repo *github.Repository) error {
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
		if mustEdit(*branch.Name) {
			protect(client, repo, branch)
		}
	}

	return nil
}

func protect(client *github.Client, repo *github.Repository, branch *github.Branch) error {
	if *branch.Protected && !unprotect {
		fmt.Printf("%s: %s is already protected\n", *repo.FullName, *branch.Name)
		return nil
	}

	if !*branch.Protected && unprotect {
		fmt.Printf("%s: %s is already unprotected\n", *repo.FullName, *branch.Name)
		return nil
	}

	if !unprotect {
		fmt.Printf("%s: %s will be set to protected\n", *repo.FullName, *branch.Name)
	} else {
		fmt.Printf("%s: %s will be freed\n", *repo.FullName, *branch.Name)
	}

	if dryrun {
		return nil
	}

	activateProtection := false
	if !unprotect {
		activateProtection = true
	}
	branch.Protected = &activateProtection
	protectionReq := &github.ProtectionRequest{
		RequiredStatusChecks: nil,
		Restrictions: nil,
	}

	if _, _, err := client.Repositories.UpdateBranchProtection(*repo.Owner.Login, *repo.Name, *branch.Name, protectionReq); err != nil {
		return err
	}

	return nil
}

func mustEdit(branchName string) bool {
	for _, toProtect := range protectBranches {
		if toProtect.MatchString(branchName) {
			return true
		}
	}
	return false
}

func listRepositories(client *github.Client, startPage int) chan *github.Repository {
	result := make(chan *github.Repository, 20)
	listRepositoriesInChan(client, startPage, result)
	return result
}

func listRepositoriesInChan(client *github.Client, startPage int, result chan *github.Repository) {
	opt := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{
			Page:    startPage,
			PerPage: 20,
		},
	}

	repos, resp, err := client.Repositories.List("", opt)
	if err != nil {
		log.Println(err)
		close(result)
		return
	}

	for _, repo := range repos {
		result <- repo
	}

	if startPage == resp.LastPage || resp.NextPage == 0 {
		close(result)
		return
	}

	go func() {
		listRepositoriesInChan(client, resp.NextPage, result)
	}()
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

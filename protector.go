package main

import (
	"flag"
	"fmt"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"os"
	"regexp"
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
	orgs                stringsFlag
)

type stringsFlag []string

func (s *stringsFlag) String() string {
	return fmt.Sprintf("%s", *s)
}
func (s *stringsFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type repositoriesService interface {
	ListBranches(owner string, repo string, opt *github.ListOptions) ([]*github.Branch, *github.Response, error)
	UpdateBranchProtection(owner, repo, branch string, preq *github.ProtectionRequest) (*github.Protection, *github.Response, error)
	RemoveBranchProtection(owner, repo, branch string) (*github.Response, error)
}

func main() {
	// parse flags
	flag.StringVar(&ghToken, "token", "", "GitHub API token")
	flag.BoolVar(&dryrun, "dry-run", false, "do not make any changes, just print out what would have been done")
	flag.BoolVar(&version, "version", false, "print version and exit")
	flag.BoolVar(&version, "v", false, "print version and exit (shorthand)")
	flag.BoolVar(&unprotect, "free", false, "remove branch protection")
	flag.Var(&protectRepositories, "repos", "repositories fullname to protect (ex: jcgay/maven-color)")
	flag.Var(&orgs, "orgs", "organizations name to protect")

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

	if len(orgs) > 0 && len(protectRepositories) > 0 {
		usageAndExit("Can't filter repositories by name and organization at the same time", 1)
	}

	protectBranches = make([]*regexp.Regexp, 0)
	for _, branch := range branches {
		protectBranches = append(protectBranches, regexp.MustCompile(branch))
	}

	if len(protectBranches) == 0 {
		protectBranches = append(protectBranches, regexp.MustCompile("^master$"))
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghToken},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	client := github.NewClient(tc)

	var ghr repositories
	if len(protectRepositories) > 0 {
		ghr = &selectedGitHubRepositories{
			client:        client,
			selectedRepos: protectRepositories,
		}
	} else if len(orgs) > 0 {
		ghr = &orgsGitHubRepositories{
			client: client,
			orgs:   orgs,
		}
	} else {
		ghr = &allGitHubRepositories{
			client: client,
		}
	}

	repos := ghr.fetch()

	gp := &githubProtection{
		repositoriesService: client.Repositories,
		branchPatterns:      protectBranches,
		successOutput:       os.Stdout,
		failureOutput:       os.Stderr,
	}

	var wg sync.WaitGroup
	for repo := range repos {
		wg.Add(1)
		go func(repository *github.Repository) {
			defer wg.Done()

			if unprotect {
				gp.free(repository)
			} else {
				gp.protect(repository)
			}
		}(repo)
	}
	wg.Wait()

	os.Exit(0)
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

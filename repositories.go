package main

import (
	"github.com/google/go-github/github"
	"log"
	"strings"
	"sync"
)

type repositories interface {
	fetch() chan *github.Repository
}

type allGitHubRepositories struct {
	client *github.Client
}

func (aghr *allGitHubRepositories) fetch() chan *github.Repository {
	result := make(chan *github.Repository, 20)
	aghr.list(1, result)
	return result
}

func (aghr *allGitHubRepositories) list(startPage int, result chan *github.Repository) {
	opt := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{
			Page:    startPage,
			PerPage: 20,
		},
	}

	repos, resp, err := aghr.client.Repositories.List("", opt)
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
		aghr.list(resp.NextPage, result)
	}()
}

type selectedGitHubRepositories struct {
	client        *github.Client
	selectedRepos []string
}

func (sghr *selectedGitHubRepositories) fetch() chan *github.Repository {
	result := make(chan *github.Repository)
	var wg sync.WaitGroup
	for _, repoFullName := range sghr.selectedRepos {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			metas := strings.SplitN(name, "/", 2)
			if repo, _, err := sghr.client.Repositories.Get(metas[0], metas[1]); err == nil {
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

type orgsGitHubRepositories struct {
	client *github.Client
	orgs   []string
}

func (aghr *orgsGitHubRepositories) fetch() chan *github.Repository {
	result := make(chan *github.Repository, 20)
	var wg sync.WaitGroup
	for _, orga := range orgs {
		wg.Add(1)
		go func(orga string) {
			defer wg.Done()
			aghr.listByOrg(1, orga, result)
		}(orga)
	}

	go func() {
		wg.Wait()
		close(result)
	}()

	return result
}

func (aghr *orgsGitHubRepositories) listByOrg(startPage int, orga string, result chan *github.Repository) {
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{
			Page:    startPage,
			PerPage: 20,
		},
	}

	repos, resp, err := aghr.client.Repositories.ListByOrg(orga, opt)
	if err != nil {
		log.Println(err)
		return
	}

	for _, repo := range repos {
		result <- repo
	}

	if startPage == resp.LastPage || resp.NextPage == 0 {
		return
	}

	aghr.listByOrg(resp.NextPage, orga, result)
}

package main

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	"io"
	"net/http"
	"regexp"
)

type protection interface {
	protect(repo *github.Repository)
	free(repo *github.Repository)
}

type githubProtection struct {
	repositoriesService repositoriesService
	branchPatterns      []*regexp.Regexp
	successOutput       io.Writer
	failureOutput       io.Writer
}

type success string
type failure string

func (gp *githubProtection) process(repo *github.Repository, modify func(*github.Branch) (success, failure)) {
	if (*repo.Permissions)["admin"] == false {
		fmt.Fprintf(gp.failureOutput, "%s: you don't have admin rights to modify this repository\n", *repo.FullName)
		return
	}

	branches, err := gp.filterBranches(repo)
	if err != nil {
		fmt.Fprint(gp.failureOutput, err)
	}

	for _, branch := range branches {
		success, failure := modify(branch)
		if failure != "" {
			fmt.Fprintln(gp.failureOutput, failure)
		} else {
			fmt.Fprintln(gp.successOutput, success)
		}
	}
}

func (gp *githubProtection) protect(repo *github.Repository) {
	gp.process(repo, func(branch *github.Branch) (success, failure) {
		return gp.lock(repo, *branch.Name)
	})
}

func (gp *githubProtection) free(repo *github.Repository) {
	gp.process(repo, func(branch *github.Branch) (success, failure) {
		return gp.unlock(repo, *branch.Name)
	})
}

func (gp *githubProtection) filterBranches(repo *github.Repository) ([]*github.Branch, error) {
	opt := &github.ListOptions{
		PerPage: 100,
	}

	branches, resp, err := gp.repositoriesService.ListBranches(context.TODO(), *repo.Owner.Login, *repo.Name, opt)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Received HTTP response [%s] when listing branches for %s", resp.Status, *repo.FullName)
	}

	result := make([]*github.Branch, 0)
	for _, branch := range branches {
		if gp.accept(*branch.Name) {
			result = append(result, branch)
		}
	}

	return result, nil
}

func withRepo(msg string, repo *github.Repository, branch *github.Branch) string {
	return fmt.Sprintf("%s: %s %s", *repo.FullName, *branch.Name, msg)
}

func (gp *githubProtection) lock(repo *github.Repository, branchName string) (success, failure) {
	branch, _, err := gp.repositoriesService.GetBranch(context.TODO(), *repo.Owner.Login, *repo.Name, branchName)
	if err != nil {
		return "", failure(withRepo(err.Error(), repo, branch))
	}

	if *branch.Protected {
		return success(withRepo("is already protected", repo, branch)), ""
	}

	if dryrun {
		return success(withRepo("will be set to protected", repo, branch)), ""
	}

	activateProtection := true
	branch.Protected = &activateProtection
	protectionReq := &github.ProtectionRequest{
		RequiredStatusChecks: nil,
		Restrictions:         nil,
	}
	if _, _, err := gp.repositoriesService.UpdateBranchProtection(context.TODO(), *repo.Owner.Login, *repo.Name, *branch.Name, protectionReq); err != nil {
		return "", failure(withRepo(err.Error(), repo, branch))
	}

	return success(withRepo("is now protected", repo, branch)), ""
}

func (gp *githubProtection) unlock(repo *github.Repository, branchName string) (success, failure) {
	branch, _, err := gp.repositoriesService.GetBranch(context.TODO(), *repo.Owner.Login, *repo.Name, branchName)
	if err != nil {
		return "", failure(withRepo(err.Error(), repo, branch))
	}

	if !*branch.Protected {
		return success(withRepo("is already unprotected", repo, branch)), ""
	}

	if dryrun {
		return success(withRepo("will be freed", repo, branch)), ""
	}

	if _, err := gp.repositoriesService.RemoveBranchProtection(context.TODO(), *repo.Owner.Login, *repo.Name, *branch.Name); err != nil {
		return "", failure(withRepo(err.Error(), repo, branch))
	}

	return success(withRepo("is now free", repo, branch)), ""
}

func (gp *githubProtection) accept(branchName string) bool {
	for _, toProtect := range gp.branchPatterns {
		if toProtect.MatchString(branchName) {
			return true
		}
	}
	return false
}

package main

import (
	"bytes"
	"context"
	"github.com/google/go-github/github"
	"net/http"
	"regexp"
	"testing"
)

type TestProtectRepositoryMock struct{}

func (p *TestProtectRepositoryMock) ListBranches(ctx context.Context, owner string, repo string, opt *github.ListOptions) ([]*github.Branch, *github.Response, error) {
	name := "branche-1"
	branch1 := &github.Branch{
		Name:      &name,
		Protected: nil,
	}

	resp := &github.Response{
		Response: &http.Response{
			StatusCode: 200,
		},
	}

	return []*github.Branch{branch1}, resp, nil
}

func (p *TestProtectRepositoryMock) UpdateBranchProtection(ctx context.Context, owner, repo, branch string, preq *github.ProtectionRequest) (*github.Protection, *github.Response, error) {
	return nil, nil, nil
}

func (p *TestProtectRepositoryMock) RemoveBranchProtection(ctx context.Context, owner, repo, branch string) (*github.Response, error) {
	return nil, nil
}

func (p *TestProtectRepositoryMock) GetBranch(ctx context.Context, owner, repo, branchName string) (*github.Branch, *github.Response, error) {
	notProtected := false
	branch := &github.Branch{
		Name:      &branchName,
		Protected: &notProtected,
	}
	return branch, nil, nil
}

func TestProtectRepository(t *testing.T) {
	// Given
	success := new(bytes.Buffer)
	failure := new(bytes.Buffer)

	gp := githubProtection{
		repositoriesService: &TestProtectRepositoryMock{},
		branchPatterns:      []*regexp.Regexp{regexp.MustCompile("^branch")},
		successOutput:       success,
		failureOutput:       failure,
	}

	repoName := "maven-color"
	login := "jcgay"
	repoFullName := login + "/" + repoName
	repository := &github.Repository{
		Name:     &repoName,
		FullName: &repoFullName,
		Owner:    &github.User{Login: &login},
		Permissions: &map[string]bool{
			"admin": true,
		}}

	// When
	gp.protect(repository)

	// Then
	if failure.String() != "" {
		t.Errorf("Was not expecting a failure, got: [%s]", failure.String())
	}

	if success.String() != "jcgay/maven-color: branche-1 is now protected\n" {
		t.Errorf("The repository should be locked with a success message, got: [%s]", success.String())
	}
}

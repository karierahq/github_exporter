package exporter

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v66/github"
)

func closeBody(resp *github.Response) {
	if resp != nil {
		resp.Body.Close()
	}
}

func alreadyCollected(collected []string, needle string) bool {
	for _, val := range collected {
		if needle == val {
			return true
		}
	}

	return false
}

func boolToFloat64(val bool) float64 {
	if val {
		return 1.0
	}

	return 0.0
}

func reposByOwnerAndName(ctx context.Context, client *github.Client, owner, repo string, perPage int) ([]*github.Repository, error) {
	if strings.Contains(repo, "*") {
		opts := &github.SearchOptions{
			ListOptions: github.ListOptions{
				PerPage: perPage,
			},
		}

		var (
			repos []*github.Repository
		)

		for {
			result, resp, err := client.Search.Repositories(
				ctx,
				fmt.Sprintf("user:%s", owner),
				opts,
			)

			if err != nil {
				closeBody(resp)
				return nil, err
			}

			repos = append(
				repos,
				result.Repositories...,
			)

			if resp.NextPage == 0 {
				closeBody(resp)
				break
			}

			closeBody(resp)
			opts.Page = resp.NextPage
		}

		return repos, nil
	}

	res, _, err := client.Repositories.Get(ctx, owner, repo)

	if err != nil {
		return nil, err
	}

	return []*github.Repository{
		res,
	}, nil
}

func branchByOwnerRepoAndName(ctx context.Context, client *github.Client, owner, repo string, branch string) ([]*github.Branch, error) {
	if strings.Contains(repo, "*") {

		var (
			branches []*github.Branch
		)

		for {
			result, resp, err := client.Repositories.GetBranch(
				ctx,
				fmt.Sprintf("user:%s", owner),
				fmt.Sprintf("repo:%s", repo),
				fmt.Sprintf("user:%s", branch),
				0,
			)

			if err != nil {
				closeBody(resp)
				return nil, err
			}

			branches = append(
				branches,
				result,
			)

			if resp.NextPage == 0 {
				closeBody(resp)
				break
			}

			closeBody(resp)
		}

		return branches, nil
	}

	res, _, err := client.Repositories.GetBranch(ctx, owner, repo, branch, 0)

	if err != nil {
		return nil, err
	}

	return []*github.Branch{
		res,
	}, nil
}

func getCommitsBetweenSHAs(ctx context.Context, client *github.Client, owner, repo, shaStart, shaEnd string) (int, error) {

	// Fetch the commit corresponding to the starting SHA to get its timestamp
	commitStart, _, err := client.Repositories.GetCommit(ctx, owner, repo, shaStart, nil)
	if err != nil {
		return 0, err
	}

	// Fetch the commit corresponding to the ending SHA to get its timestamp
	commitEnd, _, err := client.Repositories.GetCommit(ctx, owner, repo, shaEnd, nil)
	if err != nil {
		return 0, err
	}

	// Use the commit timestamps to filter commits between these two SHAs
	opt := &github.CommitsListOptions{
		Since: commitStart.GetCommit().GetCommitter().GetDate().Time,
		Until: commitEnd.GetCommit().GetCommitter().GetDate().Time,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	totalCommits := 0

	// Loop through paginated results and count commits
	for {
		commits, resp, err := client.Repositories.ListCommits(ctx, owner, repo, opt)
		if err != nil {
			return 0, err
		}

		totalCommits += len(commits)

		// Check if there are more pages of commits
		if resp.NextPage == 0 {
			break
		}

		// Move to the next page
		opt.Page = resp.NextPage
	}

	return totalCommits, nil
}

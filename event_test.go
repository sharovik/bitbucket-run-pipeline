package bitbucketrunpipeline

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFindAllPullRequestsInText(t *testing.T) {
	var cases = map[string][]PullRequest{
		"run pipeline for https://bitbucket.org/sharovik/test-cases/pull-requests/1": {
			{
				ID:             1,
				Workspace:      "sharovik",
				RepositorySlug: "test-cases",
			},
		},
		"run pipeline for https://bitbucket.org/sharovik/test-cases/pull-requests/1 https://bitbucket.org/sharovik2/test-cases2/pull-requests/2": {
			{
				ID:             1,
				Workspace:      "sharovik",
				RepositorySlug: "test-cases",
			},
			{
				ID:             2,
				Workspace:      "sharovik2",
				RepositorySlug: "test-cases2",
			},
		},
	}

	for text, expectedList := range cases {
		actual, err := findAllPullRequestsInText(text)
		assert.NoError(t, err)
		assert.Len(t, actual, len(expectedList))
		assert.Equal(t, expectedList, actual)
	}
}

func TestExtractInfoFromString(t *testing.T) {
	type TestCase struct {
		PullRequests []PullRequest
		Repositories []string
		Pipeline     string
	}

	var cases = map[string]TestCase{
		"run test https://bitbucket.org/sharovik/test-cases/pull-requests/1 and for repository my-repository https://bitbucket.org/sharovik/test-cases/pull-requests/2": {
			PullRequests: []PullRequest{
				{
					ID:             1,
					Workspace:      "sharovik",
					RepositorySlug: "test-cases",
				},
				{
					ID:             2,
					Workspace:      "sharovik",
					RepositorySlug: "test-cases",
				},
			},
			Repositories: []string{
				"my-repository",
			},
			Pipeline: "test",
		},
		"start test https://bitbucket.org/sharovik/test-cases/pull-requests/1 and for repository my-repository and repository my-repository2 https://bitbucket.org/sharovik/test-cases/pull-requests/2": {
			PullRequests: []PullRequest{
				{
					ID:             1,
					Workspace:      "sharovik",
					RepositorySlug: "test-cases",
				},
				{
					ID:             2,
					Workspace:      "sharovik",
					RepositorySlug: "test-cases",
				},
			},
			Repositories: []string{
				"my-repository",
				"my-repository2",
			},
			Pipeline: "test",
		},
	}

	for text, testCase := range cases {
		pullRequests, pipeline, repositories, err := extractInfoFromString(text)
		assert.NoError(t, err)
		assert.Len(t, pullRequests, len(testCase.PullRequests))
		assert.Len(t, repositories, len(testCase.Repositories))
		assert.Equal(t, testCase.PullRequests, pullRequests)
		assert.Equal(t, testCase.Repositories, repositories)
		assert.Equal(t, testCase.Pipeline, pipeline)
	}
}

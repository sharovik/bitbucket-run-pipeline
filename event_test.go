package bitbucket_run_pipeline

import (
	"errors"
	"github.com/sharovik/devbot/internal/container"
	"github.com/sharovik/devbot/internal/dto"
	"github.com/sharovik/devbot/internal/log"
	mock "github.com/sharovik/devbot/test/mock/client"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"runtime"
	"testing"
)

func init() {
	//We switch pointer to the root directory for control the path from which we need to generate test-data file-paths
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "../../")
	_ = os.Chdir(dir)
	log.Init(log.Config(container.C.Config))
}

func TestBbRunPipelineEvent_Execute(t *testing.T) {
	//PullRequest status OPEN but no participants
	container.C.BibBucketClient = &mock.MockedBitBucketClient{
		IsTokenInvalid: true,
		PullRequestInfoResponse: dto.BitBucketPullRequestInfoResponse{
			Title:        "Some title",
			Description:  "Feature;Some task description;\\(https://some-url.net/browse/error-502\\);JohnDoeProject",
			State:        pullRequestStateOpen,
			Source: dto.Source{
				Branch: struct {
					Name string `json:"name"`
				}{
					"test",
				},
			},
		},
		RunPipelineResponse: dto.BitBucketResponseRunPipeline{
			BuildNumber: 11,
		},
	}

	var msg = dto.SlackRequestChatPostMessage{
		OriginalMessage: dto.SlackResponseEventMessage{
			Text: `start staging-deploy https://bitbucket.org/john/test-repo/pull-requests/1/testing-pr-flow`,
		},
	}

	answer, err := Event.Execute(msg)
	assert.NoError(t, err)

	expectedText := "Done. Here the link to the build status report: https://bitbucket.org/john/test-repo/addon/pipelines/home#!/results/11"
	assert.Equal(t, expectedText, answer.Text)
}

func TestBbRunPipelineEvent_ExecuteNoPullRequestAndPipeline(t *testing.T) {
	//PullRequest status OPEN but no participants
	container.C.BibBucketClient = &mock.MockedBitBucketClient{
		IsTokenInvalid: true,
		PullRequestInfoError: errors.New("Bad pull-request info response "),
		RunPipelineError: errors.New("Bad pipeline response"),
	}

	var msg = dto.SlackRequestChatPostMessage{
		OriginalMessage: dto.SlackResponseEventMessage{
			Text: `start`,
		},
	}

	answer, err := Event.Execute(msg)
	assert.NoError(t, err)

	expectedText := "Could you please tell me which pipeline I should run?"
	assert.Equal(t, expectedText, answer.Text)
}

func TestBbRunPipelineEvent_ExecuteNoPipeline(t *testing.T) {
	//PullRequest status OPEN but no participants
	container.C.BibBucketClient = &mock.MockedBitBucketClient{
		IsTokenInvalid: true,
		PullRequestInfoError: errors.New("Bad pull-request info response "),
		RunPipelineError: errors.New("Bad pipeline response"),
	}

	var msg = dto.SlackRequestChatPostMessage{
		OriginalMessage: dto.SlackResponseEventMessage{
			Text: `start https://bitbucket.org/john/test-repo/pull-requests/1/testing-pr-flow`,
		},
	}

	answer, err := Event.Execute(msg)
	assert.Error(t, err)

	expectedText := "Failed to extract the data from your message"
	assert.Equal(t, expectedText, answer.Text)
}

func TestBbRunPipelineEvent_ExecuteNoPullRequest(t *testing.T) {
	//PullRequest status OPEN but no participants
	container.C.BibBucketClient = &mock.MockedBitBucketClient{
		IsTokenInvalid: true,
		PullRequestInfoError: errors.New("Bad pull-request info response "),
		RunPipelineError: errors.New("Bad pipeline response"),
	}

	var msg = dto.SlackRequestChatPostMessage{
		OriginalMessage: dto.SlackResponseEventMessage{
			Text: `start test`,
		},
	}

	answer, err := Event.Execute(msg)
	assert.NoError(t, err)

	expectedText := "Could you please tell me which pipeline I should run?"
	assert.Equal(t, expectedText, answer.Text)
}

func TestBbRunPipelineEvent_ExecuteBadPullRequestLink(t *testing.T) {
	//PullRequest status OPEN but no participants
	container.C.BibBucketClient = &mock.MockedBitBucketClient{
		IsTokenInvalid: true,
		PullRequestInfoError: errors.New("Bad pull-request info response "),
		RunPipelineError: errors.New("Bad pipeline response"),
	}

	var msg = dto.SlackRequestChatPostMessage{
		OriginalMessage: dto.SlackResponseEventMessage{
			Text: `start deploy https://bitbucket.org/john/test-repo/pull-requests/test/testing-pr-flow`,
		},
	}

	answer, err := Event.Execute(msg)
	assert.Error(t, err)

	expectedText := "Failed to extract the data from your message"
	assert.Equal(t, expectedText, answer.Text)
}
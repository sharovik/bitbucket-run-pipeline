package bitbucketrunpipeline

import (
	"errors"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/sharovik/devbot/internal/container"
	"github.com/sharovik/devbot/internal/dto"
	"github.com/sharovik/devbot/internal/log"
	mock "github.com/sharovik/devbot/test/mock/client"
	"github.com/stretchr/testify/assert"
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
			Title:       "Some title",
			Description: "Feature;Some task description;\\(https://some-url.net/browse/error-502\\);JohnDoeProject",
			State:       pullRequestStateOpen,
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

	var msg = dto.BaseChatMessage{
		OriginalMessage: dto.BaseOriginalMessage{
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
		IsTokenInvalid:       true,
		PullRequestInfoError: errors.New("Bad pull-request info response "),
		RunPipelineError:     errors.New("Bad pipeline response"),
	}

	var msg = dto.BaseChatMessage{
		OriginalMessage: dto.BaseOriginalMessage{
			Text: `start`,
		},
	}

	answer, err := Event.Execute(msg)
	assert.NoError(t, err)

	expectedText := "Sorry, please specify pipeline and pull-request/repository, because I cannot understand what to do."
	assert.Equal(t, expectedText, answer.Text)
}

func TestBbRunPipelineEvent_ExecuteNoPipeline(t *testing.T) {
	//PullRequest status OPEN but no participants
	container.C.BibBucketClient = &mock.MockedBitBucketClient{
		IsTokenInvalid:       true,
		PullRequestInfoError: errors.New("Bad pull-request info response "),
		RunPipelineError:     errors.New("Bad pipeline response"),
	}

	var msg = dto.BaseChatMessage{
		OriginalMessage: dto.BaseOriginalMessage{
			Text: `start https://bitbucket.org/john/test-repo/pull-requests/1/testing-pr-flow`,
		},
	}

	answer, err := Event.Execute(msg)
	assert.NoError(t, err)

	expectedText := "Sorry, please specify pipeline and pull-request/repository, because I cannot understand what to do."
	assert.Equal(t, expectedText, answer.Text)
}

func TestBbRunPipelineEvent_ExecuteNoPullRequest(t *testing.T) {
	//PullRequest status OPEN but no participants
	container.C.BibBucketClient = &mock.MockedBitBucketClient{
		IsTokenInvalid:       true,
		PullRequestInfoError: errors.New("Bad pull-request info response "),
		RunPipelineError:     errors.New("Bad pipeline response"),
	}

	var msg = dto.BaseChatMessage{
		OriginalMessage: dto.BaseOriginalMessage{
			Text: `start test`,
		},
	}

	answer, err := Event.Execute(msg)
	assert.NoError(t, err)

	expectedText := "Sorry, please specify pipeline and pull-request/repository, because I cannot understand what to do."
	assert.Equal(t, expectedText, answer.Text)
}

func TestBbRunPipelineEvent_ExecuteRepositoryInsteadPullRequestLink(t *testing.T) {
	container.C.Config.BitBucketConfig.DefaultMainBranch = "master"
	container.C.Config.BitBucketConfig.DefaultWorkspace = "test-workspace"

	//We received an error during the pipeline execution
	container.C.BibBucketClient = &mock.MockedBitBucketClient{
		IsTokenInvalid:   true,
		RunPipelineError: errors.New("Bad pipeline response"),
	}

	var msg = dto.BaseChatMessage{
		OriginalMessage: dto.BaseOriginalMessage{
			Text: `start test repository my-repo`,
		},
	}

	answer, err := Event.Execute(msg)
	assert.Error(t, err)

	expectedText := "I tried to run selected pipeline `test` for pull-request `#0` and I failed. Here is the reason: ```Bad pipeline response```"
	assert.Equal(t, expectedText, answer.Text)

	container.C.BibBucketClient = &mock.MockedBitBucketClient{
		IsTokenInvalid:   true,
		RunPipelineError: nil,
	}

	msg = dto.BaseChatMessage{
		OriginalMessage: dto.BaseOriginalMessage{
			Text: `start test repository my-repo`,
		},
	}

	answer, err = Event.Execute(msg)
	assert.NoError(t, err)

	expectedText = "Done. Here the link to the build status report: https://bitbucket.org/test-workspace/my-repo/addon/pipelines/home#!/results/0"
	assert.Equal(t, expectedText, answer.Text)

	container.C.BibBucketClient = &mock.MockedBitBucketClient{
		IsTokenInvalid:   true,
		RunPipelineError: nil,
	}

	msg = dto.BaseChatMessage{
		OriginalMessage: dto.BaseOriginalMessage{
			Text: `start test my-repo`,
		},
	}

	answer, err = Event.Execute(msg)
	assert.NoError(t, err)

	expectedText = "Sorry, please specify pipeline and pull-request/repository, because I cannot understand what to do."
	assert.Equal(t, expectedText, answer.Text)
}

func TestBbRunPipelineEvent_ExecuteBadPullRequestLink(t *testing.T) {
	//PullRequest status OPEN but no participants
	container.C.BibBucketClient = &mock.MockedBitBucketClient{
		IsTokenInvalid:       true,
		PullRequestInfoError: errors.New("Bad pull-request info response "),
		RunPipelineError:     errors.New("Bad pipeline response"),
	}

	var msg = dto.BaseChatMessage{
		OriginalMessage: dto.BaseOriginalMessage{
			Text: `start deploy https://bitbucket.org/john/test-repo/pull-requests/test/testing-pr-flow`,
		},
	}

	answer, err := Event.Execute(msg)
	assert.NoError(t, err)

	expectedText := "For which repository I need to run `deploy` pipeline?"
	assert.Equal(t, expectedText, answer.Text)
}

func TestBbRunPipelineEvent_ExecuteWithCustomText(t *testing.T) {
	container.C.Config.BitBucketConfig.DefaultMainBranch = "master"
	container.C.Config.BitBucketConfig.DefaultWorkspace = "test-workspace"

	container.C.BibBucketClient = &mock.MockedBitBucketClient{
		IsTokenInvalid:   true,
		RunPipelineError: nil,
	}

	var msg = dto.BaseChatMessage{
		OriginalMessage: dto.BaseOriginalMessage{
			Text: `start deploy pipeline for repository test`,
		},
	}

	answer, err := Event.Execute(msg)
	assert.NoError(t, err)

	expectedText := "Done. Here the link to the build status report: https://bitbucket.org/test-workspace/test/addon/pipelines/home#!/results/0"
	assert.Equal(t, expectedText, answer.Text)

	msg = dto.BaseChatMessage{
		OriginalMessage: dto.BaseOriginalMessage{
			Text: `Hey, can you start deploy pipe for repository test?`,
		},
	}

	answer, err = Event.Execute(msg)
	assert.NoError(t, err)

	expectedText = "Done. Here the link to the build status report: https://bitbucket.org/test-workspace/test/addon/pipelines/home#!/results/0"
	assert.Equal(t, expectedText, answer.Text)
}

func TestBbRunPipelineEvent_ExecuteHelp(t *testing.T) {
	//PullRequest status OPEN but no participants
	container.C.BibBucketClient = &mock.MockedBitBucketClient{
		IsTokenInvalid:       true,
		PullRequestInfoError: errors.New("Bad pull-request info response "),
		RunPipelineError:     errors.New("Bad pipeline response"),
	}

	var msg = dto.BaseChatMessage{
		OriginalMessage: dto.BaseOriginalMessage{
			Text: `start --help`,
		},
	}

	answer, err := Event.Execute(msg)
	assert.NoError(t, err)
	assert.Equal(t, helpMessage, answer.Text)
}

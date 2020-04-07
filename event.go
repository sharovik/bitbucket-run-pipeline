package bitbucket_run_pipeline

import (
	"errors"
	"fmt"
	"github.com/sharovik/devbot/internal/container"
	"github.com/sharovik/devbot/internal/dto"
	"github.com/sharovik/devbot/internal/log"
	"regexp"
	"strconv"
	"strings"
)

const (
	//EventName the name of the event
	EventName = "bitbucket_run_pipeline"

	//EventVersion the version of the event
	EventVersion = "1.0.0"

	pullRequestsRegex = `(?im)(start)(?:\s?)(.+)(?:\s?)(https:\/\/bitbucket.org\/(?P<workspace>.+)\/(?P<repository_slug>.+)\/pull-requests\/(?P<pull_request_id>\d+)?)`
	pullRequestStateOpen   = "OPEN"

	pipelineRefTypeBranch = "branch"
	pipelineTargetTypePipelineRefTarget = "pipeline_ref_target"
	pipelineSelectorTypeCustom = "custom"
	pipelineRegex = `(?i)([a-z_-]+)`
)

//PullRequest the pull-request item
type PullRequest struct {
	ID             int64
	RepositorySlug string
	Workspace      string
	Title          string
	Description    string
	Branch         string
}

//BbRunPipelineEvent the struct for the event object
type BbRunPipelineEvent struct {
	EventName string
}

//Event - object which is ready to use
var Event = BbRunPipelineEvent{
	EventName: EventName,
}

//Execute method which is called by message processor
func (e BbRunPipelineEvent) Execute(message dto.SlackRequestChatPostMessage) (dto.SlackRequestChatPostMessage, error) {
	pipeline, receivedPullRequest, err := extractPullRequestFromSubject(pullRequestsRegex, message.OriginalMessage.Text)
	if err != nil {
		message.Text = "Failed to extract the data from your message"
		return message, err
	}

	if pipeline == "" {
		message.Text = "Could you please tell me which pipeline I should run?"
		return message, nil
	}

	emptyPullRequest := PullRequest{}
	if receivedPullRequest == emptyPullRequest {
		message.Text = "Please define the pull-request, because I don't understand for which branch I need to run it"
		return message, nil
	}

	info, err := container.C.BibBucketClient.PullRequestInfo(receivedPullRequest.Workspace, receivedPullRequest.RepositorySlug, receivedPullRequest.ID)
	if err != nil {
		message.Text = "Failed to get the info from the API about selected pull-request"
		return message, err
	}

	replacer := strings.NewReplacer("\\", "")
	receivedPullRequest.Title = info.Title
	receivedPullRequest.Description = replacer.Replace(info.Description)
	receivedPullRequest.Branch = info.Source.Branch.Name

	response, err := container.C.BibBucketClient.RunPipeline(receivedPullRequest.Workspace, receivedPullRequest.RepositorySlug, dto.BitBucketRequestRunPipeline{
		Target: dto.PipelineTarget{
			RefName:  receivedPullRequest.Branch,
			RefType:  pipelineRefTypeBranch,
			Selector: dto.PipelineTargetSelector{
				Type: pipelineSelectorTypeCustom,
				Pattern: pipeline,
			},
			Type:     pipelineTargetTypePipelineRefTarget,
		},
	})

	if err != nil {
		message.Text = fmt.Sprintf("I tried to run selected pipeline `%s` for branch `%s` and I failed. Reason: %s", pipeline, receivedPullRequest.Branch, err.Error())
		return message, err
	}

	buildUrl := fmt.Sprintf("https://bitbucket.org/%s/%s/addon/pipelines/home#!/results/%d", receivedPullRequest.Workspace, receivedPullRequest.RepositorySlug, response.BuildNumber)
	message.Text = fmt.Sprintf("Done. Here the link to the build status report: %s", buildUrl)

	return message, nil
}

//Install method for installation of event
func (e BbRunPipelineEvent) Install() error {
	log.Logger().Debug().
		Str("event_name", EventName).
		Str("event_version", EventVersion).
		Msg("Start event Install")
	eventID, err := container.C.Dictionary.FindEventByAlias(EventName)
	if err != nil {
		log.Logger().AddError(err).Msg("Error during FindEventBy method execution")
		return err
	}

	if eventID == 0 {
		log.Logger().Info().
			Str("event_name", EventName).
			Str("event_version", EventVersion).
			Msg("Event wasn't installed. Trying to install it")

		eventID, err := container.C.Dictionary.InsertEvent(EventName, EventVersion)
		if err != nil {
			log.Logger().AddError(err).Msg("Error during FindEventBy method execution")
			return err
		}

		log.Logger().Debug().
			Str("event_name", EventName).
			Str("event_version", EventVersion).
			Int64("event_id", eventID).
			Msg("Event installed")

		scenarioID, err := container.C.Dictionary.InsertScenario(EventName, eventID)
		if err != nil {
			return err
		}

		log.Logger().Debug().
			Str("event_name", EventName).
			Str("event_version", EventVersion).
			Int64("scenario_id", scenarioID).
			Msg("Scenario installed")

		questionID, err := container.C.Dictionary.InsertQuestion("start", "Ok, give me a min", scenarioID, "(?i)(start)", "")
		if err != nil {
			return err
		}

		log.Logger().Debug().
			Str("event_name", EventName).
			Str("event_version", EventVersion).
			Int64("question_id", questionID).
			Msg("Question installed")
	}

	return nil
}

//Update for event update actions
func (e BbRunPipelineEvent) Update() error {
	return nil
}

func extractPullRequestFromSubject(regex string, subject string) (string, PullRequest, error) {
	re, err := regexp.Compile(regex)

	if err != nil {
		log.Logger().AddError(err).Msg("Error during the Find Matches operation")
		return "", PullRequest{}, err
	}

	match := re.FindStringSubmatch(subject)
	result := PullRequest{}

	if match == nil {
		log.Logger().Info().Msg("There is no match")
		return "", result, nil
	}

	var pipeline = ""
	if match[2] == "" {
		return "", PullRequest{}, errors.New("Pipeline cannot be empty ")
	}

	pipeline, err = cleanPipelineName(strings.TrimSpace(match[2]))
	if err != nil {
		return "", PullRequest{}, err
	}

	if pipeline == "" {
		return "", PullRequest{}, errors.New("Pipeline cannot be empty ")
	}

	if match[3] == "" || match[6] == "" {
		return "", PullRequest{}, errors.New("Could not parse the pull-request properly ")
	}

	item := PullRequest{}
	item.Workspace = match[4]
	item.RepositorySlug = match[5]
	item.ID, err = strconv.ParseInt(match[6], 10, 64)
	if err != nil {
		log.Logger().AddError(err).
			Interface("match", match).
			Msg("Error during pull-request ID parsing")
		return "", PullRequest{}, err
	}

	result = item

	return pipeline, result, nil
}

func cleanPipelineName(pipeline string) (string, error) {
	re, err := regexp.Compile(pipelineRegex)
	if err != nil {
		log.Logger().AddError(err).Msg("Error during the Find Matches operation")
		return "", err
	}

	match := re.FindStringSubmatch(pipeline)
	if match == nil {
		return "", nil
	}

	return match[0], nil
}
package bitbucketrunpipeline

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sharovik/devbot/internal/helper"

	"github.com/sharovik/devbot/internal/container"
	"github.com/sharovik/devbot/internal/dto"
	"github.com/sharovik/devbot/internal/log"
)

const (
	//EventName the name of the event
	EventName = "bitbucket_run_pipeline"

	//EventVersion the version of the event
	EventVersion         = "1.0.0"
	pullRequestsRegex    = `(?im)(start)(?:\s?)(.+)(?:\s?)(https:\/\/bitbucket.org\/(?P<workspace>.+)\/(?P<repository_slug>.+)\/pull-requests\/(?P<pull_request_id>\d+)?)`
	pullRequestStateOpen = "OPEN"

	helpMessage = "Send me message ```start {YOUR_CUSTOM_PIPELINE} {BITBUCKET_PULL_REQUEST_URL}``` to run the pipeline for selected pull-request."

	pipelineRefTypeBranch               = "branch"
	pipelineTargetTypePipelineRefTarget = "pipeline_ref_target"
	pipelineSelectorTypeCustom          = "custom"
	pipelineRegex                       = `(?i)([a-z_-]+)`
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
func (e BbRunPipelineEvent) Execute(message dto.BaseChatMessage) (dto.BaseChatMessage, error) {
	isHelpAnswerTriggered, err := helper.HelpMessageShouldBeTriggered(message.OriginalMessage.Text)
	if err != nil {
		log.Logger().Warn().Err(err).Msg("Something went wrong with help message parsing")
	}

	if isHelpAnswerTriggered {
		message.Text = helpMessage
		return message, nil
	}

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
			RefName: receivedPullRequest.Branch,
			RefType: pipelineRefTypeBranch,
			Selector: dto.PipelineTargetSelector{
				Type:    pipelineSelectorTypeCustom,
				Pattern: pipeline,
			},
			Type: pipelineTargetTypePipelineRefTarget,
		},
	})

	if err != nil {
		message.Text = fmt.Sprintf("I tried to run selected pipeline `%s` for branch `%s` and I failed. Reason: %s", pipeline, receivedPullRequest.Branch, err.Error())
		return message, err
	}

	buildURL := fmt.Sprintf("https://bitbucket.org/%s/%s/addon/pipelines/home#!/results/%d", receivedPullRequest.Workspace, receivedPullRequest.RepositorySlug, response.BuildNumber)
	message.Text = fmt.Sprintf("Done. Here the link to the build status report: %s", buildURL)

	if container.C.Config.BitBucketConfig.ReleaseChannelMessageEnabled && container.C.Config.BitBucketConfig.ReleaseChannel != "" {
		log.Logger().Debug().
			Str("channel", container.C.Config.BitBucketConfig.ReleaseChannel).
			Msg("Send release-confirmation message")

		response, statusCode, err := container.C.SlackClient.SendMessage(dto.SlackRequestChatPostMessage{
			Channel:           container.C.Config.BitBucketConfig.ReleaseChannel,
			Text:              fmt.Sprintf("The user <@%s> asked me to run `%s` pipeline for a branch `%s`. Here the link to build-report: %s", message.OriginalMessage.User, pipeline, receivedPullRequest.Branch, buildURL),
			AsUser:            true,
			Ts:                time.Time{},
			DictionaryMessage: dto.DictionaryMessage{},
			OriginalMessage:   dto.SlackResponseEventMessage{},
		})

		if err != nil {
			log.Logger().AddError(err).
				Interface("response", response).
				Interface("status", statusCode).
				Msg("Failed to sent answer message")
		}
	}

	return message, nil
}

//Install method for installation of event
func (e BbRunPipelineEvent) Install() error {
	log.Logger().Debug().
		Str("event_name", EventName).
		Str("event_version", EventVersion).
		Msg("Triggered event installation")

	return container.C.Dictionary.InstallEvent(
		EventName,           //We specify the event name which will be used for scenario generation
		EventVersion,        //This will be set during the event creation
		"start",             //Actual question, which system will wait and which will trigger our event
		"Ok, give me a min", //Answer which will be used by the bot
		"(?i)(start)",       //Optional field. This is regular expression which can be used for question parsing.
		"",                  //Optional field. This is a regex group and it can be used for parsing the match group from the regexp result
	)
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

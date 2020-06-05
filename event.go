package bitbucketrunpipeline

import (
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
	EventVersion = "1.0.1"

	initRegex        = `(?im)(start)(?:\s+)([a-z-_]+)`
	pullRequestRegex = `(https:\/\/bitbucket.org\/(?P<workspace>.+)\/(?P<repository_slug>.+)\/pull-requests\/(?P<pull_request_id>\d+)?)`
	repositoryRegex  = `((?:repository)\s+([a-z-_]+))`

	pullRequestStateOpen = "OPEN"

	helpMessage = "Send me message ```start {YOUR_CUSTOM_PIPELINE} {BITBUCKET_PULL_REQUEST_URL}``` to run the pipeline for selected pull-request."

	pipelineRefTypeBranch               = "branch"
	pipelineTargetTypePipelineRefTarget = "pipeline_ref_target"
	pipelineSelectorTypeCustom          = "custom"
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

var (
	//Event - object which is ready to use
	Event = BbRunPipelineEvent{
		EventName: EventName,
	}
)

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

	matches, err := compileRegex(message.OriginalMessage.Text)
	if err != nil {
		message.Text = fmt.Sprintf("I tried to parse your text and I failed. Here why: ```%s```", err)
		return message, err
	}

	if len(matches) == 0 {
		message.Text = "Sorry, please specify pipeline and pull-request/repository, because I cannot understand what to do."
		return message, err
	}

	receivedPullRequest, err := extractPullRequest(matches)
	if err != nil {
		message.Text = fmt.Sprintf("Failed to extract pull-request data from your message. Error: ```%s```", err)
		return message, err
	}

	pipeline := extractPipeline(matches)
	if pipeline == "" {
		message.Text = "Could you please tell me which pipeline I should run?"
		return message, nil
	}

	repository := extractRepository(matches)
	if repository == "" && receivedPullRequest.ID == 0 {
		message.Text = fmt.Sprintf("For which repository I need to run `%s` pipeline?", pipeline)
		return message, nil
	}

	log.Logger().Info().
		Str("workspace", container.C.Config.BitBucketConfig.DefaultWorkspace).
		Str("main_branch", container.C.Config.BitBucketConfig.DefaultMainBranch).
		Str("pipeline", pipeline).
		Interface("pull_request", receivedPullRequest).
		Str("repository", repository).
		Msg("Run the pipeline for repository")

	var buildURL = ""
	if receivedPullRequest.ID != 0 {
		info, err := container.C.BibBucketClient.PullRequestInfo(receivedPullRequest.Workspace, receivedPullRequest.RepositorySlug, receivedPullRequest.ID)
		if err != nil {
			message.Text = fmt.Sprintf("I tried to get the info from the API about selected pull-request and I failed. Here is the reason: ```%s```", err)
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
			message.Text = fmt.Sprintf("I tried to run selected pipeline `%s` for pull-request `#%d` and I failed. Here is the reason: ```%s```", pipeline, receivedPullRequest.ID, err.Error())
			return message, err
		}

		buildURL = fmt.Sprintf("https://bitbucket.org/%s/%s/addon/pipelines/home#!/results/%d", receivedPullRequest.Workspace, receivedPullRequest.RepositorySlug, response.BuildNumber)
		message.Text = fmt.Sprintf("Done. Here the link to the build status report: %s", buildURL)
	} else {
		response, err := container.C.BibBucketClient.RunPipeline(container.C.Config.BitBucketConfig.DefaultWorkspace, repository, dto.BitBucketRequestRunPipeline{
			Target: dto.PipelineTarget{
				RefName: container.C.Config.BitBucketConfig.DefaultMainBranch,
				RefType: pipelineRefTypeBranch,
				Selector: dto.PipelineTargetSelector{
					Type:    pipelineSelectorTypeCustom,
					Pattern: pipeline,
				},
				Type: pipelineTargetTypePipelineRefTarget,
			},
		})

		if err != nil {
			message.Text = fmt.Sprintf("I tried to run selected pipeline `%s` for pull-request `#%d` and I failed. Here is the reason: ```%s```", pipeline, receivedPullRequest.ID, err.Error())
			return message, err
		}

		buildURL = fmt.Sprintf("https://bitbucket.org/%s/%s/addon/pipelines/home#!/results/%d", container.C.Config.BitBucketConfig.DefaultWorkspace, repository, response.BuildNumber)
		message.Text = fmt.Sprintf("Done. Here the link to the build status report: %s", buildURL)
	}

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

func getMainRegex() string {
	return fmt.Sprintf("%s.+(%s|%s)", initRegex, pullRequestRegex, repositoryRegex)
}

func compileRegex(subject string) (matches []string, err error) {
	var mainRegex *regexp.Regexp
	regexString := getMainRegex()
	mainRegex, err = regexp.Compile(regexString)
	if err != nil {
		log.Logger().AddError(err).Msg("Error during the Find Matches operation")
		return matches, err
	}

	matches = mainRegex.FindStringSubmatch(subject)
	if matches == nil {
		log.Logger().Info().Msg("No pull-request found")
		return matches, nil
	}

	return matches, nil
}

func extractPullRequest(matches []string) (result PullRequest, err error) {
	if matches[5] == "" || matches[6] == "" || matches[7] == "" {
		return PullRequest{}, nil
	}

	result.Workspace = matches[5]
	result.RepositorySlug = matches[6]
	result.ID, err = strconv.ParseInt(matches[7], 10, 64)
	if err != nil {
		log.Logger().AddError(err).
			Interface("matches", matches).
			Msg("Error during pull-request ID parsing")
		return PullRequest{}, err
	}

	return result, nil
}

func extractPipeline(matches []string) string {
	var pipeline string
	if matches[2] != "" {
		pipeline = matches[2]
	}

	return pipeline
}

func extractRepository(matches []string) string {
	var pipeline string
	if matches[9] != "" {
		pipeline = matches[9]
	}

	return pipeline
}

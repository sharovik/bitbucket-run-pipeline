package bitbucketrunpipeline

import (
	"fmt"
	"github.com/sharovik/devbot/internal/service/base"
	"regexp"
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
	EventVersion = "1.1.0"

	initRegex        = `(?im)(start)(?:\s+)([a-z-_]+)`
	pullRequestRegex = `(https:\/\/bitbucket.org\/(?P<workspace>.+)\/(?P<repository_slug>.+)\/pull-requests\/(?P<pull_request_id>\d+)?)`
	repositoryRegex  = `((?:repository)\s+([a-z-_]+))`

	pullRequestStateOpen = "OPEN"

	helpMessage = "Send me message ```start {YOUR_CUSTOM_PIPELINE} {BITBUCKET_PULL_REQUEST_URL}``` to run the pipeline for selected pull-request.\nOr you if you don't have the pull-request, use the repository name. ```start {YOUR_CUSTOM_PIPELINE} repository {YOUR_REPOSITORY_NAME}```. In case when you specify the repository, the default main branch will be used(for example: `master`)."

	pipelineRefTypeBranch               = "branch"
	pipelineTargetTypePipelineRefTarget = "pipeline_ref_target"
	pipelineSelectorTypeCustom          = "custom"

	defaultScenario              = "start"
	runPipelineQuestionsScenario = "run pipeline"
	defaultScenarioAnswer = "Ok, give me a min"

	stepWhatDestination = "What `pull-request` or `repository` should I use?"
	stepWhatPipeline = "What pipeline should I run?"
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

	//First let's prepare the scenarios for identification, which one is called
	defaultScenarioDM, askingScenarioDM, err := initScenarios()
	if err != nil {
		message.Text += fmt.Sprintf("\nFailed to get available scenarios. Here why: `%s`", err)
		base.DeleteConversation(message.Channel)
		return message, err
	}

	currentConversation := base.GetConversation(message.Channel)

	pipeline := ""
	repository := ""
	receivedPullRequest := PullRequest{}
	if currentConversation.ScenarioID != int64(0) {
		switch currentConversation.ScenarioID {
		case defaultScenarioDM.ScenarioID:
			receivedPullRequest, pipeline, repository, err = extractInfoFromString(message.OriginalMessage.Text)
			if err != nil {
				message.Text = fmt.Sprintf("Failed to extract data from your message. Error: ```%s```", err)
				message.Text += "\nProbably you don't use the correct message template.\n"
				message.Text += helpMessage
				base.DeleteConversation(message.Channel)
				return message, err
			}
			break
		case askingScenarioDM.ScenarioID:
			if len(currentConversation.Variables) != 2 {
				message.Text += "\nFor some reason I received not all answers. Please repeat, what do you want?"
				base.DeleteConversation(message.Channel)
				return message, nil
			}

			receivedPullRequest, pipeline, repository, err = extractInfoFromConversationVariables(currentConversation.Variables)
			if err != nil {
				message.Text += "\nHm.. For some reason I cannot parse received information. It looks like some of the variables does not have proper value.\n"
				message.Text += "\nPlease check the message template using `start --help`"
				base.DeleteConversation(message.Channel)
				return message, err
			}
			break
		}
	} else {
		receivedPullRequest, pipeline, repository, err = extractInfoFromString(message.OriginalMessage.Text)
		if err != nil {
			message.Text = "Hmm.. I don't understand what and where need to execute. Probably you didn't used the correct message template. Ask me `start --help` for more info.\n"
			message.Text += "Anyway, I will ask you now couple of questions.\n"
			message.Text += fmt.Sprintf("\n\n%s", stepWhatDestination)

			base.DeleteConversation(message.Channel)
			base.AddConversation(message.Channel, askingScenarioDM.QuestionID, dto.BaseChatMessage{
				Channel:           message.Channel,
				Text:              stepWhatDestination,
				AsUser:            false,
				Ts:                time.Now(),
				DictionaryMessage: askingScenarioDM,
				OriginalMessage:   dto.BaseOriginalMessage{
					Text:  message.OriginalMessage.Text,
					User:  message.OriginalMessage.User,
					Files: message.OriginalMessage.Files,
				},
			}, "")
			return message, nil
		}
	}

	log.Logger().Info().
		Str("workspace", container.C.Config.BitBucketConfig.DefaultWorkspace).
		Str("main_branch", container.C.Config.BitBucketConfig.DefaultMainBranch).
		Str("pipeline", pipeline).
		Interface("pull_request", receivedPullRequest).
		Str("repository", repository).
		Msg("Run the pipeline for repository")

	var (
		buildURL       = ""
		selectedBranch = container.C.Config.BitBucketConfig.DefaultMainBranch
	)

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

		selectedBranch = receivedPullRequest.Branch
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
			message.Text = fmt.Sprintf("I tried to run selected pipeline `%s` for `%s` and I failed. Here is the reason: ```%s```", pipeline, repository, err.Error())
			return message, err
		}

		buildURL = fmt.Sprintf("https://bitbucket.org/%s/%s/addon/pipelines/home#!/results/%d", container.C.Config.BitBucketConfig.DefaultWorkspace, repository, response.BuildNumber)
		message.Text = fmt.Sprintf("Done. Here the link to the build status report: %s", buildURL)
	}

	if container.C.Config.BitBucketConfig.ReleaseChannelMessageEnabled && container.C.Config.BitBucketConfig.ReleaseChannel != "" {
		log.Logger().Debug().
			Str("channel", container.C.Config.BitBucketConfig.ReleaseChannel).
			Msg("Send release-confirmation message")

		response, statusCode, err := container.C.MessageClient.SendMessage(dto.SlackRequestChatPostMessage{
			Channel:           container.C.Config.BitBucketConfig.ReleaseChannel,
			Text:              fmt.Sprintf("The user <@%s> asked me to run `%s` pipeline for a branch `%s`. Here the link to build-report: %s", message.OriginalMessage.User, pipeline, selectedBranch, buildURL),
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

	err := container.C.Dictionary.InstallEvent(
		EventName,           //We specify the event name which will be used for scenario generation
		EventVersion,        //This will be set during the event creation
		defaultScenario,             //Actual question, which system will wait and which will trigger our event
		defaultScenarioAnswer, //Answer which will be used by the bot
		"(?i)(start)",       //Optional field. This is regular expression which can be used for question parsing.
		"",                  //Optional field. This is a regex group and it can be used for parsing the match group from the regexp result
	)

	if err != nil {
		return err
	}

	return installAskingScenario()
}

//Update for event update actions
func (e BbRunPipelineEvent) Update() error {
	container.C.MigrationService.SetMigration(InstallationAskingScenario{})
	return container.C.MigrationService.RunMigrations()
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
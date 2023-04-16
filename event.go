package bitbucketrunpipeline

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/sharovik/devbot/internal/container"
	"github.com/sharovik/devbot/internal/database"
	"github.com/sharovik/devbot/internal/dto"
	"github.com/sharovik/devbot/internal/dto/databasedto"
	"github.com/sharovik/devbot/internal/log"
	"github.com/sharovik/devbot/internal/service"
	"github.com/sharovik/devbot/internal/service/message"
	"github.com/sharovik/devbot/internal/service/message/conversation"
	"github.com/sharovik/orm/clients"
	"github.com/sharovik/orm/query"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	//EventName the name of the event
	EventName         = "bitbucket_run_pipeline"
	VariablesScenario = "bitbucket_run_pipeline_variables"

	//EventVersion the version of the event
	EventVersion = "2.0.0"

	pipelineRegex   = `(?im)(?:start|run)(?:\s+)([a-z-_0-9]+)`
	prRegexp        = `(?im)https:\/\/bitbucket.org\/(?P<workspace>.+)\/(?P<repository_slug>.+)\/pull-requests\/(?P<pull_request_id>\d+)`
	repositoryRegex = `(?im)((?:repository)\s+([a-z-_0-9]+))`

	urlRegex = `(?m)([\w+]+\:\/\/)?([\w\d-]+\.)*[\w-]+[\.\:]\w+([\/\?\=\&\#\.]?[\w-]+)*\/?`

	helpMessage = "Send me message ```start {YOUR_CUSTOM_PIPELINE} {BITBUCKET_PULL_REQUEST_URL_1} {BITBUCKET_PULL_REQUEST_URL_2} ...{BITBUCKET_PULL_REQUEST_URL_N}``` to run the pipeline for selected pull-request.\nYou can also trigger pipeline for one or more repositories, just write ```start {YOUR_CUSTOM_PIPELINE} repository {YOUR_REPOSITORY_NAME}```. In case when you specify the repository, the default main branch will be used(for example: `master`)."

	pipelineRefTypeBranch               = "branch"
	pipelineTargetTypePipelineRefTarget = "pipeline_ref_target"
	pipelineSelectorTypeCustom          = "custom"

	defaultScenarioAnswer = "Ok, give me a min"

	stepWhatDestination = "For which `pull-requests` or `repositories` should I trigger the pipeline?"
	stepWhatPipeline    = "What pipeline should I run? Please, write the pipeline name. Eg: my-custom-pipeline"
)

// PullRequest the pull-request item
type PullRequest struct {
	ID             int64
	RepositorySlug string
	Workspace      string
	Title          string
	Description    string
	Branch         string
}

func (r PullRequest) GetURL() string {
	return fmt.Sprintf("https://bitbucket.org/%s/%s/pull-requests/%d", r.Workspace, r.RepositorySlug, r.ID)
}

// EventStruct the struct for the event object
type EventStruct struct {
}

var (
	//Event - object which is ready to use
	Event = EventStruct{}
)

func (e EventStruct) Help() string {
	return helpMessage
}

func (e EventStruct) Alias() string {
	return EventName
}

func (e EventStruct) Execute(message dto.BaseChatMessage) (dto.BaseChatMessage, error) {
	if !isAllVariablesDefined(message) {
		return triggerVariablesScenario(message)
	}

	pullRequests, pipeline, repositories, err := getVariables(message)
	if err != nil {
		log.Logger().AddError(err).Msg("Failed to extract information from this conversation")

		return message, err
	}

	log.Logger().Info().
		Str("workspace", container.C.Config.BitBucketConfig.DefaultWorkspace).
		Str("main_branch", container.C.Config.BitBucketConfig.DefaultMainBranch).
		Str("pipeline", pipeline).
		Interface("pull_requests", pullRequests).
		Interface("repositories", repositories).
		Msg("Triggered pipeline run")

	for _, pr := range pullRequests {
		if err = runPipelineForPullRequest(message, pipeline, pr); err != nil {
			log.Logger().
				AddError(err).
				Str("pipeline", pipeline).
				Interface("pull_request", pr).
				Msg("Failed to trigger pipeline for selected pull-request")
		}
	}

	for _, repository := range repositories {
		if err = runPipelineForRepository(message, pipeline, repository); err != nil {
			log.Logger().
				AddError(err).
				Str("pipeline", pipeline).
				Interface("repository", repository).
				Msg("Failed to trigger pipeline for selected repository")
		}
	}

	message.Text = "Done"

	return message, nil
}

func runPipelineForRepository(message dto.BaseChatMessage, pipeline string, repository string) error {
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
		return errors.Wrap(err, "Failed to trigger pipeline for selected repository")
	}

	buildURL := fmt.Sprintf("https://bitbucket.org/%s/%s/addon/pipelines/home#!/results/%d", container.C.Config.BitBucketConfig.DefaultWorkspace, repository, response.BuildNumber)
	_, _, err = container.C.MessageClient.SendMessage(dto.BaseChatMessage{
		Channel:           message.Channel,
		Text:              fmt.Sprintf("Pipeline `%s` for repository `%s` was triggered against `%s` branch. Here is the build url: %s", pipeline, repository, container.C.Config.BitBucketConfig.DefaultMainBranch, buildURL),
		AsUser:            true,
		Ts:                time.Now(),
		DictionaryMessage: dto.DictionaryMessage{},
		OriginalMessage:   dto.BaseOriginalMessage{},
	})
	if err != nil {
		return errors.Wrap(err, "Failed to send notification to the channel.")
	}

	return nil
}

func runPipelineForPullRequest(message dto.BaseChatMessage, pipeline string, pullRequest PullRequest) error {
	info, err := container.C.BibBucketClient.PullRequestInfo(pullRequest.Workspace, pullRequest.RepositorySlug, pullRequest.ID)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve information about this pull-pullRequest.")
	}

	replacer := strings.NewReplacer("\\", "")
	pullRequest.Title = info.Title
	pullRequest.Description = replacer.Replace(info.Description)
	pullRequest.Branch = info.Source.Branch.Name

	response, err := container.C.BibBucketClient.RunPipeline(pullRequest.Workspace, pullRequest.RepositorySlug, dto.BitBucketRequestRunPipeline{
		Target: dto.PipelineTarget{
			RefName: pullRequest.Branch,
			RefType: pipelineRefTypeBranch,
			Selector: dto.PipelineTargetSelector{
				Type:    pipelineSelectorTypeCustom,
				Pattern: pipeline,
			},
			Type: pipelineTargetTypePipelineRefTarget,
		},
	})

	if err != nil {
		_, _, err = container.C.MessageClient.SendMessage(dto.BaseChatMessage{
			Channel:           message.Channel,
			Text:              fmt.Sprintf("Failed to run `%s` pipeline for %s because of next error: ``` %s ```", pipeline, pullRequest.GetURL(), err.Error()),
			AsUser:            true,
			Ts:                time.Now(),
			DictionaryMessage: dto.DictionaryMessage{},
			OriginalMessage:   dto.BaseOriginalMessage{},
		})
		if err != nil {
			return errors.Wrap(err, "Failed to send notification to the channel.")
		}
		return errors.Wrap(err, "Failed to trigger pipeline for selected pull-request.")
	}

	buildURL := fmt.Sprintf("https://bitbucket.org/%s/%s/addon/pipelines/home#!/results/%d", pullRequest.Workspace, pullRequest.RepositorySlug, response.BuildNumber)
	_, _, err = container.C.MessageClient.SendMessage(dto.BaseChatMessage{
		Channel:           message.Channel,
		Text:              fmt.Sprintf("Pipeline `%s` for pull-request `%s` was triggered. Here is the build url: %s", pipeline, pullRequest.GetURL(), buildURL),
		AsUser:            true,
		Ts:                time.Now(),
		DictionaryMessage: dto.DictionaryMessage{},
		OriginalMessage:   dto.BaseOriginalMessage{},
	})
	if err != nil {
		return errors.Wrap(err, "Failed to send notification to the channel.")
	}

	return nil
}

func triggerVariablesScenario(msg dto.BaseChatMessage) (dto.BaseChatMessage, error) {
	scenarioID, err := getVariablesScenarioID()
	if err != nil {
		msg.Text = "Failed to trigger the main questions for the schedule scenario"
		return msg, err
	}

	//We prepare the scenario, with our event name, to make sure we execute the right at the end
	scenario, err := service.PrepareScenario(scenarioID, EventName)
	if err != nil {
		msg.Text = "Failed to get the scenario"

		return msg, err
	}

	if err = message.TriggerScenario(msg.Channel, scenario, false); err != nil {
		msg.Text = "Failed to ask scenario questions"

		return msg, err
	}

	msg.Text = ""

	return msg, nil
}

func getVariablesScenarioID() (int64, error) {
	//We are getting scenario
	q := new(clients.Query).Select(databasedto.ScenariosModel.GetColumns()).
		From(databasedto.ScenariosModel).
		Where(query.Where{
			First:    "name",
			Operator: "=",
			Second: query.Bind{
				Field: "name",
				Value: VariablesScenario,
			},
		})
	res, err := container.C.Dictionary.GetDBClient().Execute(q)
	if err != nil {
		return 0, err
	}

	if len(res.Items()) == 0 {
		return 0, errors.New("Failed to find the variables scenario")
	}

	return int64(res.Items()[0].GetField("id").Value.(int)), nil
}

func isAllVariablesDefined(message dto.BaseChatMessage) bool {
	receivedPullRequests, pipeline, repositories, err := getVariables(message)
	if err != nil {
		log.Logger().AddError(err).Msg("Failed to parse pull-request, pipeline and repository information from string.")

		return false
	}

	if pipeline == "" {
		return false
	}

	if len(receivedPullRequests) != 0 {
		return true
	}

	if len(repositories) != 0 {
		return true
	}

	return false
}

func getVariables(message dto.BaseChatMessage) (pullRequests []PullRequest, pipeline string, repositories []string, err error) {
	if conv := conversation.GetConversation(message.Channel); conv.Scenario.ID != int64(0) {
		return extractInfoFromConversationVariables(message)
	}

	return extractInfoFromString(message.OriginalMessage.Text)
}

func getAllUrls(text string) (urls []string, err error) {
	re, err := regexp.Compile(urlRegex)
	if err != nil {
		log.Logger().AddError(err).Msg("Error during the Find Matches operation")
		return
	}

	matches := re.FindAllStringSubmatch(text, -1)

	if len(matches) == 0 {
		return
	}

	for _, url := range matches {
		urls = append(urls, url[0])
	}

	return
}

func findAllPullRequestsInText(subject string) (list []PullRequest, err error) {
	urls, err := getAllUrls(subject)
	if err != nil {
		return list, errors.Wrap(err, "Failed to parse urls")
	}

	for _, url := range urls {
		re, err := regexp.Compile(prRegexp)
		if err != nil {
			return list, errors.Wrap(err, "Failed to parse pull-request")
		}

		matches := re.FindAllStringSubmatch(url, -1)

		if len(matches) == 0 {
			return list, nil
		}

		for _, id := range matches {
			if id[1] == "" {
				continue
			}

			item := PullRequest{}
			item.Workspace = id[1]
			item.RepositorySlug = id[2]
			item.ID, err = strconv.ParseInt(id[3], 10, 64)
			if err != nil {
				return list, errors.Wrap(err, "Failed to parse pull-request ID")
			}

			list = append(list, item)
		}
	}

	return list, nil
}

func getFromVariable(message dto.BaseChatMessage, variableQuestion string) (value string) {
	conv := conversation.GetConversation(message.Channel)

	//If we already have opened conversation, we will try to get the answer from the required variables
	if conv.Scenario.ID != int64(0) {
		for _, variable := range conv.Scenario.RequiredVariables {
			if variableQuestion == variable.Question {
				return strings.TrimSpace(variable.Value)
			}
		}
	}

	return ""
}

// Install method for installation of event
func (e EventStruct) Install() error {
	log.Logger().Debug().
		Str("event_name", EventName).
		Str("event_version", EventVersion).
		Msg("Triggered event installation")

	err := container.C.Dictionary.InstallNewEventScenario(database.EventScenario{
		EventName:    EventName,
		ScenarioName: fmt.Sprintf("%s_initial", EventName),
		EventVersion: EventVersion,
		Questions: []database.Question{
			{
				Question:      "start",
				Answer:        defaultScenarioAnswer,
				QuestionRegex: "(?i)(start)",
			},
		},
	})

	if err != nil {
		return err
	}

	return container.C.Dictionary.InstallNewEventScenario(database.EventScenario{
		EventName:    EventName,
		ScenarioName: VariablesScenario,
		EventVersion: EventVersion,
		RequiredVariables: []database.ScenarioVariable{
			{
				Question: stepWhatDestination,
			},
			{
				Question: stepWhatPipeline,
			},
		},
	})
}

// Update for event update actions
func (e EventStruct) Update() error {
	return container.C.MigrationService.RunMigrations()
}

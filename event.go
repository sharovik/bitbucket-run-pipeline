package bitbucket_run_pipeline

import (
	"github.com/sharovik/devbot/internal/log"

	"github.com/sharovik/devbot/internal/container"
	"github.com/sharovik/devbot/internal/dto"
)

const (
	//EventName the name of the event
	EventName = "bitbucket_run_pipeline"

	//EventVersion the version of the event
	EventVersion = "1.0.0"
)

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
	message.Text = "This is an example of the answer!"
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

		questionID, err := container.C.Dictionary.InsertQuestion("Start %s", "Ok", scenarioID, "(?i)(start) (?P<pipeline>.+)", "pipeline")
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

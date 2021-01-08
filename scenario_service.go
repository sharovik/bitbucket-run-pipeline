package bitbucketrunpipeline

import (
	"fmt"
	"github.com/sharovik/devbot/internal/container"
	"github.com/sharovik/devbot/internal/dto"
	"github.com/sharovik/devbot/internal/log"
	"github.com/sharovik/devbot/internal/service"
)

//initScenarios method initialise the scenarios
func initScenarios() (defaultScenarioDM dto.DictionaryMessage, askingScenarioDM dto.DictionaryMessage, err error) {
	//First let's prepare the scenarios for identification, which one is called
	defaultScenarioDM, err = service.GenerateDMAnswerForScenarioStep(defaultScenarioAnswer)
	if err != nil {
		return
	}

	askingScenarioDM, err = service.GenerateDMAnswerForScenarioStep(stepWhatDestination)
	if err != nil {
		return
	}

	return defaultScenarioDM, askingScenarioDM, err
}

func installAskingScenario() error {
	log.Logger().Debug().
		Str("event_name", EventName).
		Str("event_version", EventVersion).
		Msg("Installing the default scenario")

	eventID, err := container.C.Dictionary.FindEventByAlias(EventName)
	if err != nil {
		return err
	}

	scenarioID, err := container.C.Dictionary.InsertScenario(fmt.Sprintf("%s: asking", EventName), eventID)
	if err != nil {
		return err
	}

	_, err = container.C.Dictionary.InsertQuestion(runPipelineQuestionsScenario, stepWhatDestination, scenarioID, "(?im)(run pipeline)", "")
	if err != nil {
		return err
	}

	_, err = container.C.Dictionary.InsertQuestion("", stepWhatPipeline, scenarioID, "", "")
	if err != nil {
		return err
	}

	if err := container.C.Dictionary.MarkMigrationExecuted(InstallationAskingScenario{}.GetName()); err != nil {
		return err
	}

	return nil
}
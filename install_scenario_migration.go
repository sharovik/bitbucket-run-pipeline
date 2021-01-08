package bitbucketrunpipeline

type InstallationAskingScenario struct {
}

func (m InstallationAskingScenario) GetName() string {
	return "installation_asking_scenario"
}

func (m InstallationAskingScenario) Execute() error {
	return installAskingScenario()
}

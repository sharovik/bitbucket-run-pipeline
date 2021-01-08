package bitbucketrunpipeline

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func extractInfoFromConversationVariables(variables []string) (receivedPullRequest PullRequest, pipeline string, repository string, err error) {
	receivedPullRequest, err = extractPullRequestFromVariable(variables[0])
	if err != nil {
		return receivedPullRequest, pipeline, repository, err
	}

	repository, err = extractRepositoryFromVariable(variables[0])
	if err != nil {
		return receivedPullRequest, pipeline, repository, err
	}

	if repository == "" {
		repository = strings.TrimSpace(variables[0])
	}

	pipeline = strings.TrimSpace(variables[1])

	return
}

func extractPullRequestFromVariable(text string) (pr PullRequest, err error) {
	regex, err := regexp.Compile(fmt.Sprintf("(?i)%s", pullRequestRegex))
	if err != nil {
		return pr, err
	}

	matches := regex.FindStringSubmatch(text)
	if matches == nil {
		return pr, nil
	}

	pr.Workspace = matches[5]
	pr.RepositorySlug = matches[6]
	pr.ID, err = strconv.ParseInt(matches[7], 10, 64)
	if err != nil {
		return pr, err
	}

	return pr, nil
}

func extractRepositoryFromVariable(text string) (repository string, err error) {
	regex, err := regexp.Compile(fmt.Sprintf("(?i)%s", repositoryRegex))
	if err != nil {
		return
	}

	matches := regex.FindStringSubmatch(text)
	if matches == nil {
		return repository, nil
	}

	repository = matches[2]
	return
}

func extractInfoFromString(text string) (receivedPullRequest PullRequest, pipeline string, repository string, err error) {
	matches, err := compileRegex(text)
	if err != nil {
		return
	}

	if len(matches) == 0 {
		err = fmt.Errorf("Failed to parse variables from the string ")
		return
	}

	receivedPullRequest, err = extractPullRequest(matches)
	if err != nil {
		return
	}

	pipeline = extractPipeline(matches)
	if pipeline == "" {
		return
	}

	repository = extractRepository(matches)
	if repository == "" && receivedPullRequest.ID == 0 {
		return
	}

	return
}

func extractPullRequest(matches []string) (result PullRequest, err error) {
	if matches[5] == "" || matches[6] == "" || matches[7] == "" {
		return PullRequest{}, nil
	}

	result.Workspace = matches[5]
	result.RepositorySlug = matches[6]
	result.ID, err = strconv.ParseInt(matches[7], 10, 64)
	if err != nil {
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
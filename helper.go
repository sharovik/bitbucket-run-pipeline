package bitbucketrunpipeline

import (
	"github.com/pkg/errors"
	"github.com/sharovik/devbot/internal/dto"
	"regexp"
	"strings"
)

func extractInfoFromConversationVariables(message dto.BaseChatMessage) (pullRequests []PullRequest, pipeline string, repositories []string, err error) {
	pipeline = getFromVariable(message, stepWhatPipeline)
	if "" == pipeline {
		return pullRequests, pipeline, repositories, errors.New("Invalid pipeline value received.")
	}

	target := getFromVariable(message, stepWhatDestination)
	if "" == target {
		return pullRequests, pipeline, repositories, errors.New("Invalid target value received.")
	}

	pullRequests, err = findAllPullRequestsInText(target)
	if err != nil {
		return pullRequests, pipeline, repositories, errors.Wrap(err, "Failed to parse pull-requests from the target string")
	}

	repositories, err = extractRepositoriesFromString(target)
	if err != nil {
		return pullRequests, pipeline, repositories, err
	}

	return pullRequests, pipeline, repositories, nil
}

func extractInfoFromString(text string) (receivedPullRequests []PullRequest, pipeline string, repositories []string, err error) {
	receivedPullRequests, err = findAllPullRequestsInText(text)
	if err != nil {
		return nil, "", nil, errors.Wrap(err, "Failed to parse pull-requests from the message")
	}

	pipeline, err = extractPipeline(text)
	repositories, err = extractRepositoriesFromString(text)
	if err != nil {
		return nil, "", nil, errors.Wrap(err, "Failed to parse repositories from the string")
	}

	return
}

func extractRepositoriesFromString(text string) (repositories []string, err error) {
	regex, err := regexp.Compile(repositoryRegex)
	if err != nil {
		return repositories, errors.Wrap(err, "Failed to create regexp object for repositories parse.")
	}

	matches := regex.FindAllStringSubmatch(text, -1)
	for _, name := range matches {
		if name[2] == "" {
			continue
		}

		repositories = append(repositories, strings.TrimSpace(name[2]))
	}

	return
}

func extractPipeline(text string) (pipeline string, err error) {
	regex, err := regexp.Compile(pipelineRegex)
	if err != nil {
		return pipeline, errors.Wrap(err, "Failed to create regexp object to parse pipeline.")
	}

	matches := regex.FindStringSubmatch(text)
	if len(matches) == 0 {
		return "", nil
	}

	pipeline = matches[1]

	return pipeline, nil
}

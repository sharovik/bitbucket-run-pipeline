# BitBucket run pipeline
Event which can run specific **custom** pipeline, selected in the message.

This is event for [sharovik/devbot](https://github.com/sharovik/devbot) automation bot.

## Table of contents
- [How it works](#how-it-works)
- [Prerequisites](#prerequisites)

## How it works
First you need to define the `pipeline`, after that you need to share the pull-request link or repository
```
Start {you-custom-pipeline-name} for http://pull-request.link
```
In that case the bot will try to get the information about the branch from the pull-request data and will use this information for pipeline run.

Let's imagine you don't have the pull-request, but you have the repository name.
``` 
Start {you-custom-pipeline-name} in repository {repository-name}
```
For this case, your text is required to have `repository {repository-name}` text.

Also you can use the bot in your custom combination:
``` 
Hey, can you start deploy-to-production for repository test?
```
## Prerequisites
Before you will start use this event please be aware of these steps

### Custom pipeline setup
In the `bitbucket-pipelines.yml` you must have defined the custom pipelines in the `pipelines` section.
Here is an example:
```yaml 
pipelines:
    custom:
        # Name of your pipeline
        deploy-staging:
              - step: *build-container
              - step: *debploy-to-staging
```

### Clone into devbot project
```
git clone git@github.com:sharovik/bitbucket-run-pipeline.git events/bitbucket_run_pipeline
```

### Install it into your devbot project
1. clone this repository into `events/` folder of your devbot project. Please make sure to use `bitbucket_run_pipeline` folder name for this event 
2. add into imports path to this event in `defined-events.go` file
``` 
import "github.com/sharovik/devbot/events/bitbucket_run_pipeline"
```
3. add this event into `defined-events.go` file to the defined events map object
``` 
DefinedEvents.Events[bitbucket_run_pipeline.EventName] = bitbucket_run_pipeline.Event
```

### Prepare environment variables in your .env
Copy and paste everything from the **#Bitbucket** section in `.env.example` file into `.env` file

### Create BitBucket client
Here [you can find how to do it](https://github.com/sharovik/devbot/blob/master/documentation/bitbucket_client_configuration.md).

# BitBucket run pipeline
Event which can run specific **custom** pipeline, selected in the message.

This is event for [sharovik/devbot](https://github.com/sharovik/devbot) automation bot.

## Table of contents
- [How it works](#how-it-works)
- [Prerequisites](#prerequisites)

## How it works
Write in PM or tag the bot user with this message
```
Start {you-custom-pipeline-name} for http://pull-request.link
```
The bot will try to run the pipeline for the branch in selected pull-request and will send you the link to the pipeline status page

## Prerequisites
Before you will start use this event please be aware of these steps

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

# BitBucket run pipeline
Event which can run specific **custom** pipeline, selected in the message.

## Table of contents
- [Prerequisites](#prerequisites)
- [Usage](#usage)

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
For this you need to do the following steps:
1. Go to your profile settings in bitbucket.org
2. Under the **ACCESS MANAGEMENT** section you will find `OAuth`, please go there
3. In the OAuth page you will find `OAuth consumers`. Please add new consumer with the following checked permissions:
- Pull requests: Read, Write
- Repositories: Read, Write
- Pipelines: Read
And also please mark this consumer as "Private".
See example of the filled form:
![Add consumer form](documentation/images/bitbucket-consumer-add-form.png)

4. After form submit you will receive the client credentials, please use them to fill these attributes in your `.env` file:
```
BITBUCKET_CLIENT_ID=
BITBUCKET_CLIENT_SECRET=
```

## Usage
Write in PM or tag the bot user with this message
```
Start {you-custom-pipeline-name}
```
You will receive in answer the pipeline, which has been triggered and where you can see the status of that pipeline
# PRMon

PRMon(itor) is a small utility which reminds your team about the open pull requests across the organization.

### Features
* Works for Enterprise git or github account
* Implemented as Lambda function which can be scheduled using cloudwatch event scheduler (Cron)
* Lambda function can be deployed using terraform.

### How to

There is only one file which can be run by providing following params
```$go
go run main.go '{"slack_web_hook_url":"https://hooks.slack.com/services/<hook url>","channel":"<channel name>","access_token":"<git access token>","base_url":"<enterpise git base url>", "git_repo_query":"search query to find repos", "git_user":"<organization name>"}'
```

#### Run it locally
To run it locally open the main.go file and uncomment the section as described in the file and comment the lambda.start line.
run go main.go to print instructions on how to run the utility. 

#### Run it as aws lambda function
To run it in aws just upload it as function and provide the argument as json 

### Deployment
To deploy it using terraform you may use the given script inside tf folder or just upload
the generated binary in zip file e.g. main.zip using following command
```go
aws lambda create-function --region us-west-1 --function-name prmon --zip-file fileb://./main.zip --runtime go1.x --handler main --profile <your configured aws profile> --role <iam aws role> 
```

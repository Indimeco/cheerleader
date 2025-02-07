`cheerleader` is a zero/low cost, run-it-yourself leaderboard service.

If you have a game you want a leaderboard for, but you don't want to pay to run it on EC2, and you don't want to use somebody else's service that may or may not exist tomorrow, then think of this as your personal `cheerleader`. Go build what you want; we're rooting for you!

# How it works

`cheerleader` utilizes low/zero cost AWS infrastructure which is included as part of the AWS free tier. You will pay nothing to run cheerleader unless you are running other services that use your free tier quota or your game becomes a sensational hit and exceeds free tier limits (which is the best we can hope for, right?).

The architecture is composed of a single API Gateway endpoint which proxies all requests to a single lambda. This lambda handles all the different API paths and methods. Data is stored in DynamoDB. Logs are only retained for a few days.

# Deployment

To deploy your own leaderboard service you will need to
1. install terraform
1. install go
1. install make
1. install and configure the aws cli
1. clone cheerleader
1. `make deploy`
1. copy the api url from your terminal output
1. build your game using the cheerleader [api](./openapi.yaml)

# Deletion

It is easy to completely remove cheerleader from your AWS account
1. `make destroy`

# Development

Perform code changes
1. `make build`
1. `make apply`

## Run tests

Cheerleader runs integration tests against a local DynamoDB

1. install docker
1. `go test ./...`

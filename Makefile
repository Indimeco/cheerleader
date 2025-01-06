build:
	GOOS=linux GOARCH=arm64 go build -v -ldflags '-d -s' -a -tags netgo -installsuffix netgo -o build/bin/app .

init:
	terraform init infra

plan:
	terraform plan infra

apply:
	terraform apply --auto-approve infra

destroy:
	terraform destroy --auto-approve infra

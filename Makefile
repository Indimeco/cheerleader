.DEFAULT_GOAL := build

.PHONY:fmt vet build

clean:
	go clean

fmt: clean
	go fmt ./...

vet: fmt
	go vet ./...

build:
	GOOS=linux GOARCH=amd64 go build -o build/bootstrap .

init:
	terraform init infra

plan:
	terraform plan infra

apply:
	terraform apply --auto-approve infra

destroy:
	terraform destroy --auto-approve infra

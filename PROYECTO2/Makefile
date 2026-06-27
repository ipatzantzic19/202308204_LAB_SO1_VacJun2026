SHELL := /usr/bin/env bash

.PHONY: proto test build push images deploy validate zot-provision

proto:
	./scripts/generate-proto.sh

test:
	./scripts/test.sh

build:
	./scripts/images.sh build

push:
	./scripts/images.sh push

images:
	./scripts/images.sh all

deploy:
	./scripts/deploy.sh

validate:
	./scripts/validate.sh

zot-provision:
	./infra/zot/provision-gcp.sh

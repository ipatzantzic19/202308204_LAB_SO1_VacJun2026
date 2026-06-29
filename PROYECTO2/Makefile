SHELL := /usr/bin/env bash

.PHONY: proto test build push images artifact artifact-pull deploy validate load-test validate-hpa zot-provision kubevirt

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

artifact:
	./scripts/oci-artifact.sh publish

artifact-pull:
	./scripts/oci-artifact.sh pull

deploy:
	./scripts/deploy.sh

validate:
	./scripts/validate.sh

load-test:
	./scripts/run-load-tests.sh

validate-hpa:
	./scripts/validate-hpa.sh

kubevirt:
	./scripts/install-kubevirt.sh

zot-provision:
	./infra/zot/provision-gcp.sh

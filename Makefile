.PHONY: clean version help run-bomb-squad images
.DEFAULT_GOAL=all
#.NOTPARALLEL: deps

GOCMD=go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
SHORT_SHA := $(shell git rev-parse --short HEAD)
BOMB_SQUAD_DIR = .
BOMB_SQUAD_UPTODATE = $(BOMB_SQUAD_DIR)/.uptodate
BOMB_SQUAD_IMG = bomb-squad:$(SHORT_SHA)

PROM_VERSION := $(shell grep -E '^prometheus \d\.\d\.\d$$' VERSION | awk '{ print $$2 }')
PROM_RULES_VERSION := $(shell grep -E '^prometheus-rules \d\.\d\.\d$$' VERSION | awk '{ print $$2 }')
BOMB_SQUAD_VERSION := $(shell grep -E '^bomb-squad \d\.\d\.\d$$' VERSION | awk '{ print $$2 }')

BOMB_SQUAD_FILES := $(shell find $(BOMB_SQUAD_DIR) -type f -name '*.go' -print)

IMAGE_NAME := gcr.io/freshtracks-io/bomb-squad:$(SHORT_SHA)

#Gopkg.lock: Gopkg.toml
#	dep ensure
#	@touch Gopkg.lock
#
#deps: Gopkg.lock ## Ensure the dependencies are up-to-date

version:
	@echo PROMETHEUS: $(PROM_VERSION)
	@echo PROMETHEUS RULES: $(PROM_RULES_VERSION)
	@echo BOMB SQUAD: $(BOMB_SQUAD_VERSION)

$(BOMB_SQUAD_UPTODATE): $(BOMB_SQUAD_FILES)
	docker build \
	  --build-arg PROM_VERSION=$(PROM_VERSION) \
		--build-arg PROM_RULES_VERSION=$(PROM_RULES_VERSION) \
		--build-arg BOMB_SQUAD_VERSION=$(BOMB_SQUAD_VERSION) \
		--file $(BOMB_SQUAD_DIR)/Dockerfile \
		--tag $(BOMB_SQUAD_IMG) \
		. \
		&& touch $(BOMB_SQUAD_UPTODATE)

build: $(BOMB_SQUAD_UPTODATE) ## Docker-based build of relevant exes

bomb-squad: $(BOMB_SQUAD_UPTODATE) ## Build local bomb-squad image
run-bomb-squad: ## Run local bomb-squad image. To pass args, use `make run-bomb-squad ARGS="arg1=val1,..."`
	@docker run -it $(BOMB_SQUAD_IMG) $(ARGS)

all: build ## Build all the things

clean: ## Remove binaries and docker images
	rm -f $(BOMB_SQUAD_UPTODATE)
	@docker rmi --force $(IMAGE_NAME) 2>/dev/null

help: ## This help text
	@awk -F ':|##' '/^[^\t].+?:.*?##/ {printf "\033[36m%-30s\033[0m %s\n", $$1, $$NF}' $(MAKEFILE_LIST) | sort

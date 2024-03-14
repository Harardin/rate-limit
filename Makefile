SHELL            := /bin/sh
GOBIN            ?= $(GOPATH)/bin
PATH             := $(GOBIN):$(PATH)
GO               = go
TARGET_DIR       ?= $(PWD)/.build


ifeq ($(DELVE_ENABLED),true)
GCFLAGS	= -gcflags 'all=-N -l'
endif

# Setting path to installed go tools if $GOPATH is empty
ifeq ($(GOPATH),)
GOBIN ?=$(GOBIN)
endif


.PHONY: start
start:
	CONSUL_STAND_NAME=local go run ./cmd/app/main.go

test:
	go test ./...

.PHONY: build
build:
	$(info $(M) building application...)
	@GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(GCFLAGS) $(LDFLAGS) -o $(TARGET_DIR)/cmd ./cmd/*.go

.PHONY: watch
watch: ## Run binaries that rebuild themselves on changes
	$(info $(M) run...)
	@$(GOBIN)/air -c $(PWD)/.air.conf

upgrade:
	GOWORK=off go-mod-upgrade
	go mod tidy

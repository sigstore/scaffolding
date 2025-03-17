# optionally set these env variables
FULCIO_REPO ?= https://github.com/sigstore/fulcio.git
REKOR_REPO ?= https://github.com/sigstore/rekor.git

# prevent a conflict with rekor-server's metrics port.
export FULCIO_METRICS_PORT = 2113

CHECKOUT_DIR := ./CHECKOUT_DIR
FULCIO_DIR := $(CHECKOUT_DIR)/fulcio
REKOR_DIR := $(CHECKOUT_DIR)/rekor

.PHONY: all
all: checkout up

.PHONY: up
up:
# bring up both fulcio and rekor simultaneously.
	make up-fulcio up-rekor -j 2

.PHONY: down
down:
# bring down both fulcio and rekor simultaneously.
	make down-fulcio down-rekor -j 2

.PHONY: up-fulcio
up-fulcio:
	docker compose -f $(FULCIO_DIR)/docker-compose.yml up --wait

.PHONY: down-fulcio
down-fulcio:
	docker compose -f $(FULCIO_DIR)/docker-compose.yml down

.PHONY: up-rekor
up-rekor:
	docker compose -f $(REKOR_DIR)/docker-compose.yml up --wait

.PHONY: down-rekor
down-rekor:
	docker compose -f $(REKOR_DIR)/docker-compose.yml down

.PHONY: checkout
checkout:
# checkout the repos locally.
ifeq ($(wildcard $(FULCIO_DIR)),)
	git clone --depth 1 $(FULCIO_REPO) $(FULCIO_DIR)
else
	@echo "Already cloned fulcio"
endif
ifeq ($(wildcard $(REKOR_DIR)),)
	git clone --depth 1 $(REKOR_REPO) $(REKOR_DIR)
else
	@echo "Already cloned rekor"
endif

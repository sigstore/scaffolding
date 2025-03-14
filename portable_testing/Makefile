# optionally set these env variables git clone --depth 1 $(REKOR_REPO) $(CHECKOUT_DIR)/rekor
REKOR_REPO ?= git@github.com:sigstore/rekor.git
FULCIO_REPO ?= git@github.com:sigstore/fulcio.git

CHECKOUT_DIR := ./CHECKOUT_DIR
FULCIO_DIR := $(CHECKOUT_DIR)/fulcio
REKOR_DIR := $(CHECKOUT_DIR)/rekor
	

clone:
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

up:
	docker compose -f $(FULCIO_DIR)/docker-compose.yml up -d
	docker compose -f $(REKOR_DIR)/docker-compose.yml up -d
	
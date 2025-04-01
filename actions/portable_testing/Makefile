# Copyright 2025 The Sigstore Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# optionally set these env variables
FULCIO_REPO ?= https://github.com/sigstore/fulcio.git
REKOR_REPO ?= https://github.com/sigstore/rekor-tiles.git

# prevent a conflict with rekor-server's metrics port.
export FULCIO_METRICS_PORT = 2113

CHECKOUT_DIR ?= ./CHECKOUT_DIR
FULCIO_DIR ?= $(CHECKOUT_DIR)/fulcio
REKOR_DIR ?= $(CHECKOUT_DIR)/rekor

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
	cd $(FULCIO_DIR) && docker compose up --wait

.PHONY: down-fulcio
down-fulcio:
	cd $(FULCIO_DIR) && docker compose down

.PHONY: up-rekor
up-rekor:
	cd $(REKOR_DIR) && docker compose up --wait

.PHONY: down-rekor
down-rekor:
	cd $(REKOR_DIR) && docker compose down

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

.PHONY: clean
clean:
	rm -rf $(FULCIO_DIR) $(REKOR_DIR)

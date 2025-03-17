# Portable Sigstore Testing Containers

Launch Testing containers of Rekor and Fulcio withing Github Actions workflows, or run the included Makefile locally with `make up`.

It will clone the rekor and fulcio repos and launch their respective docker-compse.yml's.

There is also a work-in-progress docker-docmpose.yml that is meant to be launched with `docker compose up`.

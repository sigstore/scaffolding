# Portable Sigstore Testing Containers

Launch Testing containers of Rekor and Fulcio withing Github Actions workflows, or run the included Makefile locally with `make up`.

It will clone the rekor and fulcio repos and launch their respective docker-compse.yml's.

> [!IMPORTANT]
> If you use git ssh URLs for `FULCIO_REPO` and `REKOR_REPO`, and you're using a `actions/checkout` step before calling this the Action, you will need to add `persist-credentials: true`.

There is also a work-in-progress docker-docmpose.yml that is meant to be launched with `docker compose up`.

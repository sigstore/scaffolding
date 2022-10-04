# Scaffolding Release Process

## Prerequisites

You should be part of the `scaffolding-codeowners` or `sigstore-oncall` groups, which are defined within the
[community repo](https://github.com/sigstore/community/blob/main/github-sync/github-data/users.yaml).

## Steps

1. Visit <https://github.com/sigstore/scaffolding/releases/new>
1. Click on the `Choose a tag` drop-down
1. For the tag name, use a new version number, for instance: v0.4.11
1. Click on the `+ Create new tag: v0.4.11 on publish` button that appears.
1. For the release title, use the tag name: v0.4.11,
1. Click the `Generate release notes` button
1. Click the `Save Draft` button to trigger the `Release` action.
1. Monitor the [`Release` action](https://github.com/sigstore/scaffolding/actions/workflows/release.yaml), which generates the remaining release artifacts
1. Once the `Release` action has been completed successfully, find your draft release at the [releases page](https://github.com/sigstore/scaffolding/releases)
1. Click the green "Publish release" button

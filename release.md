# Scaffolding Release Process

## Prerequisites

You should be part of the `scaffolding-codeowners` or `sigstore-oncall` groups, which are defined within the
[community repo](https://github.com/sigstore/community/blob/main/github-sync/github-data/users.yaml).

## Steps

### Sync tags

Ensure that sure your local branch is up-to-date from the upstream:

```shell
git pull upstream main --tags
```

### Pick a new version number

The scaffolding repo uses [semver](https://semver.org/). Your first step is to determine the latest tag used.

List the latest tags in date order:

```shell
git tag | tail
```

Example output:

```
...
v0.0.0
v0.1.0
```

Show a list of changes since the latest version (v0.1.0):

```shell
git log v0.1.0..
```

If the commits include a new feature or breaking change, bump the minor version. If it only includes bug fixes, bump the patch version.

### Tagging

Once you have a version number in mind, tag it locally:

```shell
git tag -a v0.2.0 -m v0.2.0
```

Then push the tag upstream:

```shell
git push upstream v0.2.0
```

### Monitor the release automation

Once the tag is pushed, the [`Release` action](https://github.com/sigstore/scaffolding/actions/workflows/release.yaml) will generate the appropriate release artifacts and create a draft release.

Be sure to see how long recent runs have taken. At the time of this writing, the release job takes 30 to 40 minutes to execute.

### Publish

Once the `Release` action has been completed successfully:

1. Find your draft release on the [releases page](https://github.com/sigstore/scaffolding/releases)
2. Click the `Generate release notes` button to fill in the "What's Changed" section
3. Click the green `Publish release` button
4. 🎉

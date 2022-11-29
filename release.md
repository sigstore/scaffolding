# Scaffolding Release Process

## Prerequisites

You should be part of the `scaffolding-codeowners` or `sigstore-oncall` groups, which are defined within the
[community repo](https://github.com/sigstore/community/blob/main/github-sync/github-data/users.yaml).

## Steps

### Make sure your local branch is up-to-date from the upstream, for example:

```shell
git pull upstream main --tags
```

### Verify the correct version you're about to tag:

```shell
git tag
```

This will list all the tags, so the latest (at the time of this writing), was:
`v0.5.1`

```shell
vaikas@villes-mbp scaffolding % git tag
v0.1-alpha
v0.1.1-alpha
v0.1.10-alpha
v0.1.11-alpha
<SNIPPED FOR READABILITY>
v0.4.9
v0.5.0
v0.5.1
```

So, I will release `v0.5.2`

### Tag the release with the new version number, for instance `v0.5.2`

```shell
git tag -a v0.5.2 -m "v0.5.2"
```

### Push the tag

```shell
git push upstream v0.5.2
```

### Monitor the [`Release` action](https://github.com/sigstore/scaffolding/actions/workflows/release.yaml), which generates the remaining release artifacts

### Once the `Release` action has been completed successfully, find your draft release at the [releases page](https://github.com/sigstore/scaffolding/releases)

### Update the release notes by clicking on the `Generate release notes` button

### Click the green "Publish release" button

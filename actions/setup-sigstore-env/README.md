# Sigstore Testing Containers

Launch Testing containers of Rekor and Fulcio withing Github Actions workflows, or run the included [./run-containers.sh](./run-containers.sh) on your local machine.

It will clone the rekor and fulcio repos and launch their respective docker-compse.yml files.

## Local Use

The script will export env variables you may need.

```shell
rm signing_config.json trusted_root.json
source ./run-containers.sh
```

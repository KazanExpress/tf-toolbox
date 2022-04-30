# tf-toolbox

Tools to make terraform CI/CD process better.

## findroot

Based on changed files from git finds directory from which to run `terragrunt run-all` command.

## cleanplan

Removes duplicate whitespace from plan.

```bash
docker run --entrypoint=/bin/sh -ti -v $PWD:/mount ghcr.io/kazanexpress/tf-toolbox:latest

cat plan.txt | cleanplan > cleanplan.txt
```

![Repository Top Language](https://img.shields.io/github/languages/top/ivan-bokov/dstack-test)
![Github Repository Size](https://img.shields.io/github/repo-size/ivan-bokov/dstack-test)
![Github Open Issues](https://img.shields.io/github/issues/ivan-bokov/dstack-test)
![Lines of code](https://img.shields.io/tokei/lines/github/ivan-bokov/dstack-test)
![License](https://img.shields.io/badge/license-MIT-green)
![GitHub last commit](https://img.shields.io/github/last-commit/ivan-bokov/dstack-test)
![GitHub contributors](https://img.shields.io/github/contributors/ivan-bokov/dstack-test)

# dstack-test
Run a docker container with a command as a serverless and upstream the logs to aws cloudwatch with given AWS credentials.

## HOW TO
To run:
```
make run && .bin/golang-test-task --docker-image python --bash-command "echo Hello World" --cloudwatch-group golang-test-task-group-1 --cloudwatch-stream golang-test-task-group-2 --aws-access-key-id ... --aws-secret-access-key ... --aws-region ...
```
To help args
```
make run && .bin/golang-test-task --help
```
To test (for test requires local docker)
```
make test
```



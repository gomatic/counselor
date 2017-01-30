# counselor

Runs a comand after template-processing the parameters and environment with AWS
instance metadata provide as template variables.


## What it does

1. obtains the metdata from `169.254.169.254/latest/meta-data/`
1. renders template `{{.variables}}` in command-line parameters and environment variables
1. `exec` the provided command

## e.g.

    counselor run -- counseler test -- {{.LocalIp4}}

- `counselor run` runs the command `counselor test`
- `counselor test` just prints out it's parameters and its environment

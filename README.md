# counselor

[![Build Status](https://travis-ci.org/gomatic/counselor.svg?branch=master)](https://travis-ci.org/gomatic/counselor)

Runs a comand after template-processing the parameters and environment with AWS
instance metadata provide as template variables.


## What it does

1. obtains the metdata from `169.254.169.254/latest/meta-data/`.
1. renders template `{{.variables}}` in command-line parameters and environment variables.
1. `exec` the provided command.

## e.g.

Test using `/bin/echo`:

    counselor run -- /bin/echo {{.LocalIp4}}

- `counselor run` grabs all the metadata

Test using the provided debugger:

    counselor run -- counselor test -- {{.LocalIp4}}

- `counselor test` just prints out its parameters and environment

# dotenv

[![License](https://img.shields.io/github/license/FollowTheProcess/dotenv)](https://github.com/FollowTheProcess/dotenv)
[![Go Reference](https://pkg.go.dev/badge/go.followtheprocess.codes/dotenv.svg)](https://pkg.go.dev/go.followtheprocess.codes/dotenv)
[![Go Report Card](https://goreportcard.com/badge/github.com/FollowTheProcess/dotenv)](https://goreportcard.com/report/github.com/FollowTheProcess/dotenv)
[![GitHub](https://img.shields.io/github/v/release/FollowTheProcess/dotenv?logo=github&sort=semver)](https://github.com/FollowTheProcess/dotenv)
[![CI](https://github.com/FollowTheProcess/dotenv/workflows/CI/badge.svg)](https://github.com/FollowTheProcess/dotenv/actions?query=workflow%3ACI)
[![codecov](https://codecov.io/gh/FollowTheProcess/dotenv/branch/main/graph/badge.svg)](https://codecov.io/gh/FollowTheProcess/dotenv)

Load environment variables from a `.env` file

> [!WARNING]
> **dotenv is in early development and is not yet ready for use**

![caution](./docs/img/caution.png)

## Installation

```shell
go get go.followtheprocess.codes/dotenv@latest
```

## Quickstart

Given `.env` file like this:

```bash
# This is a comment and is ignored by the parser completely
NUMBER_OF_THINGS=123 # Comments can also go on lines
USERNAME=mysuperuser

# Command substitution
API_KEY=$(op read op://MyVault/SomeService/api_key)

# Variable interpolation
EMAIL=${USER}@email.com # We added $USER above
CACHE_DIR=${HOME}/.cache # Can also reference existing system env vars
DATABASE_URL="postgres://${USER}@localhost/my_database"

# Single quotes force the string to be treated as literal
# no interpolation or command substitution will happen here
LITERAL='${USER} should show up literally'

# Multiline strings can be declared with """. Leading and trailing
# whitespace will be trimmed allowing for nicer formatting.
MANY_LINES="""
This is a lot of text with multiple lines

You could use this to store the contents of a file or
an X509 cert, an SSH key etc.
"""

# Escape sequences work as you'd expect
ESCAPE_ME="Newline\n and a tab\t etc."

# You can even use the export keyword to retain compatibility with e.g. bash
export SOMETHING=yes
```

You can parse and load it from Go code like this:

```go
package main

import (
	"fmt"
	"os"

	"go.followtheprocess.codes/dotenv"
)

func main() {
	// You can also pass in a bunch of options to Load to configure
	// its behaviour like load a specific file, overwrite existing system
	// environment variables or not etc.
	if err := dotenv.Load(); err != nil {
		fmt.Fprintf("uh oh: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Success! os.Environ is now: %v\n", os.Environ)
}
```

### Credits

This package was created with [copier] and the [FollowTheProcess/go-template] project template.

[copier]: https://copier.readthedocs.io/en/stable/
[FollowTheProcess/go-template]: https://github.com/FollowTheProcess/go-template

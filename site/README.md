This directory stores the website and documentation of HTNN.

## Copyright

It uses [docsy-example](https://github.com/google/docsy-example) which is under Apache-2.0 license as the template and adds some modifications to it.
If not explicitly specified, files under this directory are under the Apache-2.0 license used by docsy-example.

## Commands

* `make build`: build the docsy image
* `make up`: start the Hugo server to review the website in real time. Then you can access the website via `http://localhost:1313/`. The docsy image should be built before running this command.
* `make clear`: remove the docsy image

## Tools

We provide some tools to maintain the site.

### cmd/translator

We use AI to translate the documentation. In details, we create some rules via prompt engineering, and let Large Language Model to translate it.

This work is done semi-automatically:

1. Run `go run cmd/translator/main.go -f content/en/path/to/x.md | pbcopy` to create prompt for translating `x.md`.
2. Find a human to submit it to LLM.
3. Tweak the output. We only need to tweak 5% of the result.

We would like to run it in CI if:

1. There is a sponsor who gives us enough money.
2. There is a GitHub Bot to submit the output as a pull request so that the reviewer can tweak it.

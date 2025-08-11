HTNN is released under the Apache 2.0 license, and follows a very standard Github development process. If you would like to contribute something, this document should help you get started and maximize the chances of your PR being merged.

## Communication

* Before starting work on a major feature, please reach out to us via GitHub, Slack,
  email, etc. We will make sure no one else is already working on it and ask you to open a
  GitHub issue.
* A "major feature" is defined as any change that is > 100 LOC altered (not including tests or generated code),
  or changes any user-facing behavior. We will use the GitHub issue to discuss the feature and come to
  agreement. This is to prevent your time being wasted, as well as ours. The GitHub review process
  for major features is also important so that maintainers can come to agreement on design.
  If it is appropriate to write a design document, the document must be hosted either in the GitHub
  tracking issue, or linked to from the issue and hosted in a public location.
* Small patches and bug fixes don't need prior communication.
* If you are going to submit a huge PR (> 300 LOC excluding generated code),
  better to split it into several PRs. Each PR should be well tested. For example,
  when submitting a new plugin, you can split it into:
  1. basic feature
  2. documentation
  3. more extra features, like configuring additional options

## Coding style

Make sure the CI is passed. This includes:

* Code formatting with `make fmt-go`
* Linting with `make lint-go` 
* License checking with `make lint-license`
* Duplicate code detection with `make lint-duplicates`

Run `make lint` to execute all linting checks, including duplicate code detection.

### Duplicate Code Detection

HTNN includes automated duplicate code detection to maintain code quality. The system will flag:
- Duplicate code blocks above 100 tokens (via golangci-lint dupl)
- Duplicate words in comments and strings (via dupword)
- Comprehensive analysis across all modules (via custom script)

If duplicates are found, consider refactoring common patterns into shared utilities or base structures. See `docs/duplicate-detection.md` for detailed guidance.

## PR review policy for maintainers

The following strategies are recommended for project maintainers to review code:

1. Check the issue with this PR
2. Check the solution's reasonability
3. Check if there are enough tests to cover the feature
4. Pay attention to the code which makes the code structure change, the error handling, the solution for the corner case and concurrency
5. Avoid breaking change, unless there is a good reason.

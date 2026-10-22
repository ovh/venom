# Security Policy

## Supported versions

Only the latest release of venom receives security fixes. Please upgrade to
the [latest release](https://github.com/ovh/venom/releases/latest) before
reporting an issue.

| Version | Supported |
|---|---|
| latest release | yes |
| older releases | no |

## Reporting a vulnerability

Please do not report security vulnerabilities through public GitHub issues,
discussions or pull requests.

Report them privately through GitHub:
[Report a vulnerability](https://github.com/ovh/venom/security/advisories/new).

Please include:

- the venom version (`venom version`) and how it was installed (binary,
  Docker image, `go install`)
- the affected component (executor, assertion, test suite parsing, ...)
- steps to reproduce, ideally a minimal test suite file
- the impact you observed or expect

## What to expect

- We acknowledge your report.
- We keep you informed of the progress towards a fix.
- Once a fix is released, we publish a GitHub security advisory and credit
  you, unless you prefer to stay anonymous.

## Scope

In scope: the venom code in this repository and the release artifacts
published on GitHub and Docker Hub (`ovhcom/venom`).

Out of scope: vulnerabilities in the systems a test suite targets, and
issues that need an attacker to already control the test suite files or the
machine running venom.

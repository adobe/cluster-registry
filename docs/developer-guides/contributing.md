# Contributing to Cluster Registry

## Welcome contributors to the project

Admit that you are eager for contributions, and we are so happy
to find you here.

### Communication Channels

The primary forum for technical and architectural discussion is GitHub issues
and pull requests. GitHub issues can be used for bug tracking, feature requests,
architectural proposals and general questions. Pull requests are used for code
review and discussion of specific proposed changes. Discussion in issues and
pull requests (PRs) should include deep context, such that new contributors in
the future can explore past discussions and fully understand the context.

## Getting Started as a Contributor

The Cluster Registry is a [Echo](https://echo.labstack.com/) microframework based
service that is used to obtain information about Kubernetes Clusters.

We recommend the following ways to start contributing:

Follow the [Local Development](./local-development.md) to run local Cluster Registry
stack on your local machine and [Architecture](./architecture.md) to understand better
the project design.

Review PRs. Even if you have no idea what the PR does! Ask basic questions about
the code, and the maintainers are happy to answer as a way of building knowledge.
Contribute to discussion of issues, features and design. Any written idea
represent a contribution! Pick some simple code contributions. Fix typos and
grammar, add tests, deduplicate and refactor small pieces of code or write
documentation. Once you've contributed in smaller ways and built familiarity
with the project, you'll be able to more easily start on larger changes.

## Planning and Considerations for Changes

Before you write a change, you should consider the following:

### Community and Consensus

Does your change have consensus and support from other contributors? Do you need
to discuss your change with others or produce a prototype demonstrator before
committing to the full change?

Larger architectural changes require a design proposal and peer review _before_
the PR can be merged. Submit a Git issue with the problem statement, and a
description of the solution you plan to build early in the process. It's
generally not productive to move beyond the prototype phase of a large change
before receiving peer feedback.

### User Impact

Will your change impact users?

### Ecosystem

Will your change impact other tools (apis) in the ecosystem?

Will your change have backwards compatibility? If no, please consider new api
schema version.

Will your change require changes in pipelines and clients such as EKO, KD, OrCA?
You may need to publish a proposal spec to coordinate
changes that cross multiple codebases and teams, and consider new api schema for
smooth migration.

Will your change require changes in observability systems such as metrics
collection, alerting, eventing, tracing or logging systems? Again, a proposal
spec be needed to coordinate changes in external systems.

### Database change

Will your change need database schema change?

Will your change introduce downtime at deployment time?

### Operations

How will your change impact end users/consumers?

Will operators need to change existing practices or adopt new ones?

Does documentation need to be created or modified?

### Security

Does your change conform to best practices around security, secrets management,
access control and encryption? How do you protect the cluster and user data
at rest and in transit?

Does your change conform to relevant compliance requirements?

### Documentation

Is your change well documented, including the _what_, _how_ and _why_ of the
change?

Do function definitions have docstrings and type hints? Do modules and scripts
have a comment near the top explaining what the code does? Do manifests have a
comment near the top with a link to their upstream sources?

Do you need to include a guide in the `docs/` directory?

### Testing

Every PR should provide evidence that the change works as expected in the
scenarios it will run. This could include automated tests, or it could include
evidence from manual testing. All PRs should include some kind of testing.

#### Unit Tests

Can parts of your code be unit tested? Can you increase test coverage by
splitting your code into multiple functions or modules? Can you increase test
coverage by parameterizing certain objects and mocking them in the unit test?

Are the tests _meaningful_? Unit tests should test documented interfaces, not
private internals. (Blackbox vs Whitebox testing)

#### Integration Tests

Do you use an integration test to check if your code is integrated with other
modules?

#### Functional Tests

Do you use a functional test to check if your code is integrated doesn't impact
Cluster Registry communication with other apis (sqs, dynamoDB)?

### External Dependencies

Will your change introduce new external dependencies? Who controls those
external systems? Is there a defined target service level? What happens when
those external systems fail? Can dependence on the external system be reduced or
eliminated?

### Code Style

Certain code style checks are automated as part of the `make go-format go-vet go-lint`
check. Automated style checks are blockers for merging. Non-automated style issues
are not blockers or merging.

When in doubt, follow the [Golang wiki page](https://github.com/golang/go/wiki).

### Code Security

Common security static checks are automated as part of the `make go-sec` check.
Medium and high severity issues are blockers for merging.


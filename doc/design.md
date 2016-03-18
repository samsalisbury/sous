# sous design

Sous is carefully designed to ensure it interacts in a useful way with both
people and machines. This document outlines in some detail what this means,
and will serve to guide the ongoing development of sous.

As new features are added, and old ones improved, this document should help us
decide whether or not a feature is appropriate, and if so, how it should work.

This document does not detail _what_ sous is, only what it should be _like._
For details on what sous actually is, please see [what is sous?]

[what is sous?]: what-is-sous.md

## Design Ethos

Sous must always follow a set of [principles, outlined below], to ensure
that it serves [the engineer], [the project], [the operations team], and
[the organisation].

[principles, outlined below]: #design-principles
[the engineer]: #for-engineers
[the project]: #for-projects
[the operations team]: #for-engineers
[the organisation]: #for-organisations

### For Engineers

Sous must always serve the engineer, not the other way around. It
provides sane defaults where they apply, but must always be configurable
per project, per infrastructure and per engineer. It must not impose any
constraints without clearly documented rationale, and must keep any such
constraints to a minimum.

Sous must never obfuscate, nor make opaque, any of its operations or
configuration. It should be instructive, and reveal its own workings to
those who are interested, but stay out of the way, and not require deep
knowledge of its own workings for those who just want to get stuff done
quickly.

### For Projects

Sous must adapt to any software project that results in  a cloud-deployed
application. It must be as broad as possible in its support of
languages, frameworks, application types, and workflows.

Sous may require that the use of languages and frameworks follow conventions
standard for that language or framework. Where a language or framework does
not provide adequate conventions on one area or another (e.g. dependency
management), sous may add features to polyfill those requirements.

### For Operations

Sous must make the job of operations teams easier by revealing and
allowing the control of applications running on their infrastructure.

Sous must always provide all its data in easy-to-use formats.

### For Organisations

Sous must help to decouple software engineering teams writing
applications and services from operations teams managing
infrastructure. It must make it easier for these two forces in
engineering to work together, solving the problems of the organisation
at their own pace, without requiring constant synchronisation.

## Design Principles

- **Speed:**
  - Sous must speed up software delivery.
  - Sous must never block progress for any reason.
  - Sous must never block deployments.
  - Sous must be monitored for its own performance as a system.
  - Sous must perform operations as quickly as the hardware will allow. (That means parallelising tasks wherever possible.)
- **Transparency:**
  - Sous must always be able to reveal its own workings, and provide
    tools to help engineers repeat them manually.
  - Sous must expose all of its data for interrogation and consumption
	by other services.
  - Sous must make visible facts about an organisation's platform that
	were previously not visible.
- **Loose Coupling:**
  - Sous must require no Sous-specific changes to application code,
	nor addition of Sous-specific files to application repositories.
	*(However the configuration of Sous, especially the contracts and buildpacks you use, may require changes. This is org-specific though. Note to organisations: think carefully before requiring org-specific things in repositories as well!)*
- **Client-Server feature parity:**
  - Every task the server is able to perform should be easy to perform locally on the command line.
  - It should be possible to work completely offline.
- **CLI first:**
  - You should be able to do _everything_ through the command line, including interacting with the server.

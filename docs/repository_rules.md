# Repository Rules

## Goal

This repository is built primarily for personal use and learning.
The goal of these rules is to keep the project clean, understandable, and easy to evolve.

## Main branch

- `main` should stay in a healthy state.
- Avoid pushing experimental or half-broken changes directly to `main`.
- Changes should normally go through a branch and a pull request.

## Branches

Use one branch per task.

Branch naming:

- `feature/<name>` for product features
- `fix/<name>` for bug fixes
- `docs/<name>` for documentation
- `refactor/<name>` for internal code improvements
- `chore/<name>` for repository setup, maintenance, tooling, or non-feature housekeeping

Examples:

- `feature/home-page`
- `feature/topic-crud`
- `docs/repository-rules`
- `chore/project-bootstrap`

## Commits

Commits should be small, meaningful, and focused on one change.

Preferred commit prefixes:

- `feat:` new functionality
- `fix:` bug fix
- `docs:` documentation
- `refactor:` internal cleanup without changing behavior
- `test:` tests
- `chore:` maintenance, setup, tooling, repository housekeeping

Examples:

- `feat: add topics page`
- `docs: add repository workflow rules`
- `chore: bootstrap project structure`

Avoid vague commit messages like:

- `fix`
- `update`
- `changes`
- `stuff`

## Pull requests

Even in a solo project, pull requests are used to keep history clean and make changes easier to review.

PR title should be short and meaningful.

PR description should usually include:

- what was done
- why it was done
- optional notes if something is intentionally postponed or limited

Do not overcomplicate PRs with unnecessary templates or ceremony.

## Scope of changes

Try not to mix unrelated changes in one branch or PR.

Good:
- one branch for repository setup
- one branch for home page
- one branch for SQLite setup

Bad:
- router + DB + CSS + import parser + training logic in one branch

## General principle

Prefer clarity over cleverness.

The repository should stay understandable for future work, not just “working right now”.
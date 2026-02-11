---
allowed-tools: Bash(git add:*), Bash(git status:*), Bash(git commit:*), Bash(git diff:*), Bash(git log:*), Bash(dotnet build:*), Bash(dotnet csharpier:*)
argument-hint: [message] | --no-verify | --amend
description: Create well-formatted commits with conventional commit format
---

# Smart Git Commit

Create well-formatted commit: $ARGUMENTS

## Current Repository State

- Git status: !git status --porcelain
- Current branch: !git branch --show-current
- Staged changes: !git diff --cached --stat
- Unstaged changes: !git diff --stat
- Recent commits: !git log --oneline -5

## What This Command Does

1. Unless specified with --no-verify, automatically runs pre-commit checks:
   - `dotnet build src/visma-common.sln --no-restore` to ensure the solution compiles
   - `dotnet csharpier --check src/` to verify formatting
2. Checks which files are staged with git status
3. If 0 files are staged, automatically adds all modified and new files with git add
4. Performs a git diff to understand what changes are being committed
5. Analyzes the diff to determine if multiple distinct logical changes are present
6. If multiple distinct changes are detected, suggests breaking the commit into multiple smaller commits
7. For each commit (or the single commit if not split), creates a commit message using conventional commit format

## Best Practices for Commits

- **Verify before committing**: Ensure code builds and passes formatting checks
- **Atomic commits**: Each commit should contain related changes that serve a single purpose
- **Split large changes**: If changes touch multiple concerns, split them into separate commits
- **Conventional commit format**: Use the format `<type>: <description>` where type is one of:
  - feat: A new feature
  - fix: A bug fix
  - docs: Documentation changes
  - style: Code style changes (formatting, etc.)
  - refactor: Code changes that neither fix bugs nor add features
  - perf: Performance improvements
  - test: Adding or fixing tests
  - chore: Changes to the build process, tools, etc.
  - ci: CI/CD related changes
  - revert: Reverting previous commits
  - wip: Work in progress
- **Present tense, imperative mood**: Write commit messages as commands (e.g., "add feature" not "added feature")
- **Concise first line**: Keep the first line under 72 characters

## Guidelines for Splitting Commits

When analyzing the diff, consider splitting commits based on these criteria:

1. **Different concerns**: Changes to unrelated parts of the codebase
2. **Different types of changes**: Mixing features, fixes, refactoring, etc.
3. **File patterns**: Changes to different types of files (e.g., source code vs documentation)
4. **Logical grouping**: Changes that would be easier to understand or review separately
5. **Size**: Very large changes that would be clearer if broken down

## Examples

Good commit messages:
- feat: add employee attendance tracking endpoint
- fix: resolve database connection leak in Recruitment API
- docs: update README with development setup instructions
- refactor: extract candidate stage logic into dedicated handler
- chore: update EF Core packages to 9.0.12
- feat: add Dapr pub/sub integration for booking events
- fix: handle null profile links in candidate creation
- ci: add conditional build for EmployeeCard service

Example of splitting commits:
- First commit: feat: add EmergencyContact entity and migration
- Second commit: feat: add CRUD handlers for EmergencyContact
- Third commit: feat: add EmergencyContact API endpoints
- Fourth commit: docs: update CHANGELOG for EmergencyContact feature

## Command Options

- --no-verify: Skip running the pre-commit checks (build, CSharpier format)
- --amend: Amend the previous commit instead of creating a new one

## Important Notes

- By default, pre-commit checks (dotnet build, dotnet csharpier) will run to ensure code quality
- If these checks fail, you'll be asked if you want to proceed with the commit anyway or fix the issues first
- If specific files are already staged, the command will only commit those files
- If no files are staged, it will automatically stage all modified and new files
- The commit message will be constructed based on the changes detected
- Before committing, the command will review the diff to identify if multiple commits would be more appropriate
- If suggesting multiple commits, it will help you stage and commit the changes separately
- Always reviews the commit diff to ensure the message matches the changes

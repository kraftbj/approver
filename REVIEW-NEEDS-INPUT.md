# Review Items Needing User Input

These items were flagged during pre-publication review and require decisions before publishing.

## A. License

Which license should the project use?

Options:
- MIT License
- Apache License 2.0
- Other

## B. README

What should the README include?

Considerations:
- Minimal (project description, install, usage) vs comprehensive (screenshots, architecture, contributing guide)
- Whether to include screenshots/GIFs of the TUI

## C. Development artifacts

Should `agent-os/` and `.claude/` directories be excluded from the public repo?

These contain development tooling configuration. Options:
- Add to `.gitignore` and remove from tracking
- Keep as-is (they document the development workflow)

## D. Cobra dependency

The project uses Cobra (`github.com/spf13/cobra`) but has no subcommands or flags. Options:
- Keep Cobra (if subcommands are planned)
- Replace with plain `main()` to reduce dependencies

## E. Screen interface

The `Screen` interface in `internal/app/screen.go` is declared but never used polymorphically (each screen type is accessed directly via struct fields). Options:
- Refactor to use the interface (store `active Screen` instead of separate fields)
- Remove the interface declaration

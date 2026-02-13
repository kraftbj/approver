# Review Fix Agent — Standards

## Architecture

- Follows existing Bubble Tea patterns: state enum, message types, tea.Cmd functions
- Fix agent reuses RunAgent from runner.go (same as review pipeline)
- Selection overlay follows the same pattern as confirm/input overlays
- Detail panel tab follows the existing detailMode cycling pattern

## Code Style

- Go conventions (gofmt, effective Go)
- Error messages match existing format: "Failed to X: %v"
- Toast messages match existing patterns (showError/showInfo + clearErrorAfter)

## Testing

- Build: `go build ./...`
- Test: `go test ./...`
- Vet: `go vet ./...`

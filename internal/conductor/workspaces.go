package conductor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const fieldSeparator = "\x1f"

// Workspace is a ready Conductor workspace that can be reused as a PR worktree.
type Workspace struct {
	RepoName      string
	RepoRoot      string
	Branch        string
	Path          string
	DirectoryName string
	Name          string
	State         string
	SessionStatus string
	UpdatedAt     string
}

// DefaultDBPath returns the default local Conductor database path on macOS.
func DefaultDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, "Library", "Application Support", "com.conductor.app", "conductor.db"), nil
}

// LoadReadyWorkspaces reads Conductor's local database, if present, and returns
// live workspaces. Missing Conductor installs and missing sqlite3 are non-fatal.
func LoadReadyWorkspaces() ([]Workspace, error) {
	dbPath, err := DefaultDBPath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("checking Conductor database: %w", err)
	}
	return LoadReadyWorkspacesFromDB(dbPath)
}

// LoadReadyWorkspacesFromDB reads ready/active workspace rows from a Conductor DB.
func LoadReadyWorkspacesFromDB(dbPath string) ([]Workspace, error) {
	sqlite, err := exec.LookPath("sqlite3")
	if err != nil {
		return nil, nil
	}

	query := `
select
  coalesce(r.name, ''),
  coalesce(r.root_path, ''),
  coalesce(w.branch, ''),
  coalesce(w.workspace_path, ''),
  coalesce(w.directory_name, ''),
  coalesce(w.workspace_name, ''),
  coalesce(w.state, ''),
  coalesce(s.status, ''),
  coalesce(w.updated_at, '')
from workspaces w
left join repos r on r.id = w.repository_id
left join sessions s on s.id = w.active_session_id
where w.state in ('ready', 'active')
  and coalesce(w.workspace_path, '') <> ''
order by datetime(w.updated_at) desc;
`
	cmd := exec.Command(sqlite, "-batch", "-noheader", "-separator", fieldSeparator, "file:"+dbPath+"?mode=ro", query)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("querying Conductor database: %w", err)
	}

	workspaces, err := ParseWorkspaceRows(string(out))
	if err != nil {
		return nil, err
	}
	return filterExistingWorkspaceDirs(workspaces), nil
}

// ParseWorkspaceRows parses sqlite3 CLI output for workspace rows.
func ParseWorkspaceRows(out string) ([]Workspace, error) {
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}

	var workspaces []Workspace
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Split(line, fieldSeparator)
		if len(fields) != 9 {
			return nil, fmt.Errorf("unexpected Conductor workspace row with %d fields", len(fields))
		}
		ws := Workspace{
			RepoName:      fields[0],
			RepoRoot:      filepath.Clean(fields[1]),
			Branch:        fields[2],
			Path:          filepath.Clean(fields[3]),
			DirectoryName: fields[4],
			Name:          fields[5],
			State:         fields[6],
			SessionStatus: fields[7],
			UpdatedAt:     fields[8],
		}
		if ws.RepoRoot == "." {
			ws.RepoRoot = ""
		}
		if ws.Path == "." {
			ws.Path = ""
		}
		workspaces = append(workspaces, ws)
	}
	return workspaces, nil
}

func filterExistingWorkspaceDirs(workspaces []Workspace) []Workspace {
	filtered := workspaces[:0]
	for _, ws := range workspaces {
		if ws.Path == "" {
			continue
		}
		info, err := os.Stat(ws.Path)
		if err != nil || !info.IsDir() {
			continue
		}
		filtered = append(filtered, ws)
	}
	return filtered
}

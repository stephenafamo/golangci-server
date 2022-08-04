package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (s *serv) linter() {
	for file := range s.files {
		if !strings.HasPrefix(file, s.rootURI) {
			s.Log.Errorf("file outisde root: %q", file)
			return
		}
		go s.lintAll()
		s.lint(file)
	}
}

func (s *serv) sendDiagnostics(result GolangCILintResult) {
	s.Log.Debugf("golangci-lint: result: %v", result)

	diagnostics := make(map[string][]protocol.Diagnostic, 0)
	for _, issue := range result.Issues {
		issue := issue

		uri := filepath.Join(s.rootURI, issue.Pos.Filename)

		//nolint:gomnd
		d := protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: issue.Pos.Line - 1, Character: issue.Pos.Column - 1},
				End:   protocol.Position{Line: issue.Pos.Line - 1, Character: issue.Pos.Column - 1},
			},
			Severity: &serverity,
			Source:   &issue.FromLinter,
			Message:  s.diagnosticMessage(&issue),
		}

		if _, ok := diagnostics[uri]; !ok {
			diagnostics[uri] = []protocol.Diagnostic{d}
		} else {
			diagnostics[uri] = append(diagnostics[uri], d)
		}
	}

	for uri, diagnostics := range diagnostics {
		if err := s.conn.Notify(
			s.ctx,
			protocol.ServerTextDocumentPublishDiagnostics,
			protocol.PublishDiagnosticsParams{
				URI:         protocol.DocumentUri(uri),
				Diagnostics: diagnostics,
			},
		); err != nil {
			s.Log.Errorf("%s", err)
		}
	}
}

var (
	command   = []string{"golangci-lint", "run", "--out-format", "json"}
	serverity = protocol.DiagnosticSeverityWarning
)

func (s *serv) lintAll() {
	if s.linting {
		return
	}

	s.linting = true
	defer func() { s.linting = false }()

	//nolint:gosec
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = uriToPath(s.rootURI)
	s.Log.Infof("running command: %s", cmd)

	b, err := cmd.Output()
	// No lint errors
	if err == nil {
		return
	}

	var result GolangCILintResult
	if err := json.Unmarshal(b, &result); err != nil {
		s.Log.Errorf("%s", err)
		return
	}

	s.Log.Debugf("golangci-lint: result: %v", result)

	s.sendDiagnostics(result)
}

func (s *serv) lint(uri string) {
	//nolint:gosec
	cmd := exec.Command(command[0],
		append(command[1:], filepath.Dir(uriToPath(uri)))...,
	)
	cmd.Dir = uriToPath(s.rootURI)
	s.Log.Infof("running command: %s", cmd)

	b, err := cmd.Output()
	// No lint errors
	if err == nil {
		if err := s.conn.Notify(
			s.ctx,
			protocol.ServerTextDocumentPublishDiagnostics,
			protocol.PublishDiagnosticsParams{
				URI: protocol.DocumentUri(uri),
			},
		); err != nil {
			s.Log.Errorf("%s", err)
		}
		return
	}

	var result GolangCILintResult
	if err := json.Unmarshal(b, &result); err != nil {
		s.Log.Errorf("%s", err)
		return
	}

	s.sendDiagnostics(result)
}

func (s *serv) diagnosticMessage(issue *Issue) string {
	return fmt.Sprintf("%s: %s", issue.FromLinter, issue.Text)
}

func uriToPath(uri string) string {
	switch {
	case strings.HasPrefix(uri, "file:///"):
		uri = uri[len("file://"):]
	case strings.HasPrefix(uri, "file://"):
		uri = uri[len("file:/"):]
	}

	if path, err := url.PathUnescape(uri); err == nil {
		uri = path
	}

	if isWindowsDriveURIPath(uri) {
		uri = strings.ToUpper(string(uri[1])) + uri[2:]
	}

	return filepath.FromSlash(uri)
}

func isWindowsDriveURIPath(uri string) bool {
	//nolint:gomnd
	if len(uri) < 4 {
		return false
	}

	return uri[0] == '/' && unicode.IsLetter(rune(uri[1])) && uri[2] == ':'
}

type Issue struct {
	FromLinter  string      `json:"FromLinter"`
	Text        string      `json:"Text"`
	SourceLines []string    `json:"SourceLines"`
	Replacement interface{} `json:"Replacement"`
	Pos         struct {
		Filename string `json:"Filename"`
		Offset   uint32 `json:"Offset"`
		Line     uint32 `json:"Line"`
		Column   uint32 `json:"Column"`
	} `json:"Pos"`
	LineRange struct {
		From uint32 `json:"From"`
		To   uint32 `json:"To"`
	} `json:"LineRange,omitempty"`
}

//nolint:unused,deadcode
type GolangCILintResult struct {
	Issues []Issue `json:"Issues"`
	Report struct {
		Linters []struct {
			Name             string `json:"Name"`
			Enabled          bool   `json:"Enabled"`
			EnabledByDefault bool   `json:"EnabledByDefault,omitempty"`
		} `json:"Linters"`
	} `json:"Report"`
}

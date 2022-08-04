package main

import (
	"context"
	"errors"
	"os/signal"
	"syscall"

	"github.com/sourcegraph/jsonrpc2"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
	"github.com/tliron/kutil/logging"

	// Must include a backend implementation. See kutil's logging/ for other options.
	_ "github.com/tliron/kutil/logging/simple"
)

const lsName = "golangci"

var version string = "0.1.0"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// This increases logging verbosity (optional)
	logging.Configure(1, nil)

	s := serv{files: make(chan string, 5)}
	s.Play(ctx) //nolint:errcheck
}

type serv struct {
	protocol.Handler
	ctx     context.Context
	conn    *jsonrpc2.Conn
	Log     logging.Logger
	rootURI string
	files   chan string

	// For project linting
	linting bool
}

func (s *serv) Play(ctx context.Context) error {
	s.Handler = protocol.Handler{
		Initialize:          s.initialize,
		Initialized:         s.initialized,
		TextDocumentDidOpen: s.textDocumentDidOpen,
		TextDocumentDidSave: s.textDocumentDidSave,
		Shutdown:            s.shutdown,
		SetTrace:            s.setTrace,
	}

	server := server.NewServer(&s.Handler, lsName, false)
	s.ctx = ctx
	s.Log = server.Log
	s.conn = server.GetStdio()

	go s.linter()

	<-ctx.Done()
	if err := s.shutdown(nil); err != nil {
		return err
	}
	s.conn.Close()

	return nil
}

func (s *serv) initialize(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	capabilities := s.Handler.CreateServerCapabilities()

	if params == nil || params.RootURI == nil {
		return nil, errors.New("No rootURI")
	}
	s.rootURI = *params.RootURI

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    lsName,
			Version: &version,
		},
	}, nil
}

func (s *serv) initialized(context *glsp.Context, params *protocol.InitializedParams) error {
	return nil
}

func (s *serv) shutdown(_ *glsp.Context) error {
	close(s.files)
	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}

func (s *serv) setTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	protocol.SetTraceValue(params.Value)
	return nil
}

func (s *serv) textDocumentDidOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	s.files <- params.TextDocument.URI
	return nil
}

func (s *serv) textDocumentDidSave(context *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
	s.files <- params.TextDocument.URI
	return nil
}

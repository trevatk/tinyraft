package main

// Example code that starts and bootstraps a single node raft
// this code also initializes a module called wordTracker used
// to communicate between the raft and application logic
//
// this code also exposes an http server with endpoints to
// interact with the raft

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	"github.com/trevatk/tinyraft"
	pb "github.com/trevatk/tinyraft/protos/raft/v1"
)

type wordTracker struct {
	mtx   sync.Mutex
	words []string
}

// interface compliance
// raft leader will call Apply when new entries are added
var _ tinyraft.Module = (*wordTracker)(nil)

func newWordTracker() *wordTracker {
	return &wordTracker{
		mtx:   sync.Mutex{},
		words: []string{},
	}
}

// Apply module implementation when receiving log payload from raft
func (wt *wordTracker) Apply(cmd []byte) error {
	wt.mtx.Lock()
	defer wt.mtx.Unlock()
	log.Printf("received command: %s", string(cmd))
	w := string(cmd)
	wt.words = append(wt.words, w)
	return nil
}

func runGrpcServer(ctx context.Context, s *grpc.Server, addr string) error {

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("net.Listen: %w", err)
	}

	// block until context is done
	// then shutdown server
	go func() {
		<-ctx.Done()
		log.Println("shutdown gRPC server")
		s.GracefulStop()
		s.Stop()
	}()

	err = s.Serve(lis)
	if err != nil {
		return fmt.Errorf("s.Serve: %w", err)
	}

	return nil
}

type wordTrackServer interface {
	postWord(w http.ResponseWriter, r *http.Request)
	getWords(w http.ResponseWriter, r *http.Request)
}

type transport struct {
	wt *wordTracker
}

// interface compliance
var _ wordTrackServer = (*transport)(nil)

func (t transport) postWord(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() { _ = r.Body.Close() }()

	c, err := tinyraft.NewCaller("127.0.0.1:50051")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = c.Apply(r.Context(), body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (t transport) getWords(w http.ResponseWriter, r *http.Request) {

	t.wt.mtx.Lock()
	defer t.wt.mtx.Unlock()

	w.WriteHeader(http.StatusAccepted)
	_, _ = fmt.Fprint(w, t.wt.words)
}

func newServeMux(wts wordTrackServer) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/word", wts.postWord)
	mux.HandleFunc("/word/", wts.getWords)
	return mux
}

func runHttpServer(ctx context.Context, handler http.Handler) error {
	s := &http.Server{
		Addr:    ":8000",
		Handler: handler,
	}

	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		errCh <- s.Shutdown(shutdownCtx)
	}()

	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen and serve: %w", err)
	}

	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	wt := newWordTracker() // custom implementation provider

	// read environmental variables to configure node
	cfg, err := tinyraft.UnmarshalConfigFromEnv()
	if err != nil {
		// never do
		// do not copy pasta
		panic(err)
	}

	// create raft server and transport
	// using provided configuration
	r, tr, err := tinyraft.NewServerAndTransport(cfg, wt)
	if err != nil {
		// never do
		// do not copy pasta
		panic(err)
	}

	grpclog.SetLoggerV2(grpclog.NewLoggerV2WithVerbosity(os.Stdout, os.Stdout, os.Stderr, 2))

	// tinyraft is built using gRPC as server transport
	// gRPC is the only dependency shipped with tiny raft
	s := grpc.NewServer()
	s.RegisterService(&pb.RaftServiceV1_ServiceDesc, tr)

	mux := newServeMux(transport{wt: wt})

	// clean up func to start/stop raft server and transport
	// since the server and transport or separate
	// both structs will need to be handled individually
	defer func() {
		cancel()

		// shutdown raft and gRPC server
		s.GracefulStop()
		_ = r.Shutdown(ctx)

		if r := recover(); r != nil {
			log.Printf("panic recover: %v", r)
		}
	}()

	// log because we can
	log.Println("start raft")
	_ = r.Start(ctx)

	log.Println("start http/1 server")
	go func() { log.Fatal(runHttpServer(ctx, mux)) }()

	log.Println("start gRPC server")
	err = runGrpcServer(ctx, s, cfg.ListenAddr)
	if err != nil {
		log.Fatalf("runGrpcServer: %v", err)
	}

	os.Exit(0)
}

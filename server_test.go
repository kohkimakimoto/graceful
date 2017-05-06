package graceful

import (
	"net"
	"net/http"
	"os"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

const (
	addr = "127.0.0.1:0"
)

func newTestListener(t *testing.T) net.Listener {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestServer_Serve(t *testing.T) {
	done := make(chan struct{}, 1)
	server := &Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		done <- struct{}{}
	})}
	l := newTestListener(t)
	defer l.Close()
	go func() {
		if err := server.Serve(l); err != nil {
			t.Errorf("server.Serve(l) => %#v; want nil", err)
		}
		done <- struct{}{}
	}()
	_, err := http.Get("http://" + l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Error("timeout")
	}

	l.Close()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Error("timeout")
	}
}

func TestServer_Serve_gracefulShutdownDefaultSignal(t *testing.T) {
	testServerServeGracefulShutdown(t)
}

func TestServer_Serve_gracefulShutdownAnotherSignal(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGHUP, syscall.SIGQUIT} {
		origSignal := ShutdownSignal
		ShutdownSignal = sig
		defer func() {
			ShutdownSignal = origSignal
		}()
		testServerServeGracefulShutdown(t)
	}
}

func testServerServeGracefulShutdown(t *testing.T) {
	done := make(chan struct{})
	server := &Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		done <- struct{}{}
	})}
	l := newTestListener(t)
	defer l.Close()
	go server.Serve(l)
	wait := make(chan struct{})
	go func() {
		if _, err := http.Get("http://" + l.Addr().String()); err != nil {
			t.Errorf("http.Get => %v; want nil", err)
		}
		wait <- struct{}{}
	}()
	<-time.After(1 * time.Second)
	pid := os.Getpid()
	p, err := os.FindProcess(pid)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Signal(ShutdownSignal); err != nil {
		t.Fatal(err)
	}
	go func() {
		<-time.After(1 * time.Second)
		if _, err = http.Get("http://" + l.Addr().String()); err == nil {
			t.Errorf("http.Get after shutdown => nil; want error")
		}
		<-done
		<-wait
		wait <- struct{}{}
	}()
	select {
	case <-wait:
	case <-time.After(5 * time.Second):
		t.Errorf("timeout")
	}
}

func TestServerState_StateStart(t *testing.T) {
	done := make(chan struct{})
	origServerState := ServerState
	ServerState = func(state State) {
		switch state {
		case StateStart:
			done <- struct{}{}
		}
	}
	defer func() {
		ServerState = origServerState
	}()
	go ListenAndServe(addr, nil)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Errorf("timeout")
	}
}

func TestServerState_StateShutdown(t *testing.T) {
	done := make(chan struct{})
	started := make(chan struct{})
	origServerState := ServerState
	ServerState = func(state State) {
		switch state {
		case StateStart:
			started <- struct{}{}
		case StateShutdown:
			done <- struct{}{}
		}
	}
	defer func() {
		ServerState = origServerState
	}()
	go ListenAndServe(addr, nil)
	select {
	case <-started:
		pid := os.Getpid()
		p, err := os.FindProcess(pid)
		if err != nil {
			t.Fatal(err)
		}
		if err := p.Signal(ShutdownSignal); err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Errorf("timeout")
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Errorf("timeout")
	}
}

func TestIsMaster(t *testing.T) {
	origEnv := make([]string, len(os.Environ()))
	copy(origEnv, os.Environ())
	for _, v := range []struct {
		env    string
		expect bool
	}{
		{FDEnvKey, false},
		{"UNKNOWN_KEY", true},
	} {
		func() {
			defer func() {
				os.Clearenv()
				for _, v := range origEnv {
					env := strings.SplitN(v, "=", 2)
					os.Setenv(env[0], env[1])
				}
			}()
			if err := os.Setenv(v.env, "1"); err != nil {
				t.Error(err)
				return
			}
			actual := IsMaster()
			expect := v.expect
			if !reflect.DeepEqual(actual, expect) {
				t.Errorf(`IsMaster() with %v=1 => %#v; want %#v`, v.env, actual, expect)
			}
		}()
	}
}

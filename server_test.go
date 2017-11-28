package netconsole

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestServerListenAndServe(t *testing.T) {
	// A combination of valid and invalid log entries are used to ensure that
	// only the valid ones are processed.
	bufs := [][]byte{
		[]byte("[   22.671488] raid6: using algorithm avx2x4 gen() 21138 MB/s"),
		[]byte("foo"),
		[]byte("[   27.151481] tpm_crb MSFT0101:00: can't request region for resource [mem 0xec06f000-0xec06ffff]"),
		[]byte("[   28.123456 broken"),
	}

	got := testServer(t, bufs)

	want := []Log{
		{
			Elapsed: 22*time.Second + 671488*time.Microsecond,
			Message: "raid6: using algorithm avx2x4 gen() 21138 MB/s",
		},
		{
			Elapsed: 27*time.Second + 151481*time.Microsecond,
			Message: "tpm_crb MSFT0101:00: can't request region for resource [mem 0xec06f000-0xec06ffff]",
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("unexpected logs (-want +got):\n%s", diff)
	}
}

func testServer(t *testing.T, bufs [][]byte) []Log {
	pc, err := net.ListenPacket("udp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Accept as many messages as indicated by input slice.
	var logsWG sync.WaitGroup
	logsWG.Add(len(bufs))

	// Gather valid logs into the slice, but decrement the waitgroup on
	// all messages so that we can ensure deterministic test output.
	var logs []Log
	s := &Server{
		handle: func(_ net.Addr, l Log) {
			defer logsWG.Done()
			logs = append(logs, l)
		},
		drop: func(_ net.Addr, _ []byte) {
			defer logsWG.Done()
		},
	}

	// Ensure serve goroutine is cleaned up.
	var serveWG sync.WaitGroup
	serveWG.Add(1)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer serveWG.Done()

		err = s.serve(ctx, pc)
		if err != nil {
			// Don't panic when shutting down.
			if strings.Contains(err.Error(), "use of closed") {
				return
			}

			panic(fmt.Sprintf("failed to serve: %v", err))
		}
	}()

	client, err := net.Dial("udp", pc.LocalAddr().String())
	if err != nil {
		t.Fatalf("failed to dial client: %v", err)
	}

	// Pass input messages to the server for processing.
	for _, b := range bufs {
		if _, err := client.Write(b); err != nil {
			t.Fatalf("failed to write message: %v", err)
		}
	}

	// Wait for logs to be processed, clean up client, clean up server,
	// then ensure goroutines stop.
	logsWG.Wait()

	_ = client.Close()

	cancel()
	_ = pc.Close()

	serveWG.Wait()

	return logs
}

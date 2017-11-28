package netconsole_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mdlayher/netconsole"
)

func TestParseLog(t *testing.T) {
	tests := []struct {
		name string
		s    string
		ll   netconsole.Log
		ok   bool
	}{
		{
			name: "empty",
			s:    "",
		},
		{
			name: "no brackets",
			s:    "   22.671488 foo",
		},
		{
			name: "bad whole number",
			s:    "[   xx.671488] foo",
		},
		{
			name: "bad decimal",
			s:    "[   22.xx] foo",
		},
		{
			name: "no message",
			s:    "[   22.671488] ",
		},
		{
			name: "OK",
			s:    "[   22.671488] raid6: using algorithm avx2x4 gen() 21138 MB/s",
			ll: netconsole.Log{
				Elapsed: 22*time.Second + 671488*time.Microsecond,
				Message: "raid6: using algorithm avx2x4 gen() 21138 MB/s",
			},
			ok: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ll, err := netconsole.ParseLog(tt.s)

			if tt.ok && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatalf("expected an error, but none occurred: %v", err)
			}

			if !tt.ok {
				// Don't bother doing comparison if log is invalid.
				t.Logf("OK error: %v", err)
				return
			}

			if diff := cmp.Diff(tt.ll, ll); diff != "" {
				t.Fatalf("unexpected Log (-want +got):\n%s", diff)
			}
		})
	}
}

func BenchmarkParseLog(b *testing.B) {
	tests := []struct {
		name string
		s    string
	}{
		{
			name: "invalid",
			s:    "foo",
		},
		{
			name: "short",
			s:    "[    0.097798] x86: Booted up 1 node, 4 CPUs",
		},
		{
			name: "long",
			s:    "[   82.742346] systemd[1]: systemd 229 running in system mode. (+PAM +AUDIT +SELINUX +IMA +APPARMOR +SMACK +SYSVINIT +UTMP +LIBCRYPTSETUP +GCRYPT +GNUTLS +ACL +XZ -LZ4 +SECCOMP +BLKID +ELFUTILS +KMOD -IDN)",
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = netconsole.ParseLog(tt.s)
			}
		})
	}
}

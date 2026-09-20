package runner

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"

	"github.com/alexistdev/geosquad/api/internal/model"
)

func TestBranchPattern(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		{"[git] branch: squad/20260919-203145", "squad/20260919-203145"},
		{"  [git]   branch:   squad/abc", "squad/abc"},
		{"[git] commit: tech lead: kontrak kerja", ""},
		{"branch: squad/tanpa-prefix", ""},
	}

	for _, c := range cases {
		m := branchPattern.FindStringSubmatch(c.line)
		got := ""
		if m != nil {
			got = m[1]
		}
		if got != c.want {
			t.Errorf("baris %q -> %q, mau %q", c.line, got, c.want)
		}
	}
}

func TestClassify(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	expired, cancelExpired := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelExpired()

	// Exit code 1 dari engine artinya SQA menolak. Itu hasil yang sah.
	exit1 := runExitCode(t, 1)

	cases := []struct {
		name       string
		ctx        context.Context
		err        error
		wantStatus model.RunStatus
		wantCode   int
	}{
		{"sukses", context.Background(), nil, model.RunPassed, 0},
		{"sqa menolak", context.Background(), exit1, model.RunFailed, 1},
		{"dibatalkan", cancelled, errors.New("signal: killed"), model.RunCancelled, -1},
		{"timeout", expired, errors.New("signal: killed"), model.RunError, -1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			status, code, _ := classify(c.ctx, c.err)
			if status != c.wantStatus {
				t.Errorf("status = %s, mau %s", status, c.wantStatus)
			}
			if code != c.wantCode {
				t.Errorf("exit code = %d, mau %d", code, c.wantCode)
			}
		})
	}
}

// runExitCode menghasilkan *exec.ExitError sungguhan, bukan tiruan, supaya
// yang diuji adalah perilaku errors.As terhadap tipe aslinya.
func runExitCode(t *testing.T, code int) error {
	t.Helper()
	cmd := exec.Command("sh", "-c", "exit "+itoa(code))
	err := cmd.Run()
	if err == nil {
		t.Fatal("perintah seharusnya gagal")
	}
	return err
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	digits := ""
	for i > 0 {
		digits = string(rune('0'+i%10)) + digits
		i /= 10
	}
	return digits
}

package winget

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type fakeRunner struct {
	calls  [][]string
	output []byte
	err    error
}

func (f *fakeRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string(nil), args...))
	if len(args) > 0 && args[0] == "--version" {
		return []byte("v1.9"), nil
	}
	return f.output, f.err
}

type codeError int

func (e codeError) Error() string { return "winget exited" }
func (e codeError) ExitCode() int { return int(e) }

func TestExactIDAndNoApplications(t *testing.T) {
	f := &fakeRunner{output: []byte("Name  Id  Version  Source\nGit  Git.Git  2.0  winget\n")}
	c := New(f)
	done, err := c.Installed(context.Background(), "Git.Git")
	if err != nil || !done {
		t.Fatalf("installed=%v error=%v", done, err)
	}
	want := []string{"list", "--id", "Git.Git", "--exact", "--source", "winget", "--disable-interactivity"}
	if !reflect.DeepEqual(f.calls[1], want) {
		t.Fatalf("args=%v", f.calls[1])
	}
	f.err = codeError(-1978335212)
	done, err = c.Installed(context.Background(), "Git.Git")
	if err != nil || done {
		t.Fatalf("not found: installed=%v error=%v", done, err)
	}
}

func TestInvalidIDDoesNotRun(t *testing.T) {
	f := &fakeRunner{}
	if err := New(f).Install(context.Background(), "Git.Git;rm"); err == nil {
		t.Fatal("invalid ID accepted")
	}
	if len(f.calls) != 0 {
		t.Fatal("ran invalid package ID")
	}
}

func TestOtherListErrorIsNotAbsence(t *testing.T) {
	f := &fakeRunner{err: errors.New("network failed")}
	if _, err := New(f).Installed(context.Background(), "Git.Git"); err == nil {
		t.Fatal("error treated as absence")
	}
}

func TestSourceAgreementRequiresUserReview(t *testing.T) {
	f := &fakeRunner{err: codeError(-1978335162)}
	c := New(f)
	installed, err := c.Installed(context.Background(), "Git.Git")
	if installed || err == nil || !strings.Contains(err.Error(), "winget list --id Git.Git --exact --source winget") {
		t.Fatalf("installed=%v error=%v", installed, err)
	}
	if err := c.Install(context.Background(), "Git.Git"); err == nil || !strings.Contains(err.Error(), "review and accept") {
		t.Fatalf("install error=%v", err)
	}
	want := []string{"install", "--id", "Git.Git", "--exact", "--source", "winget", "--scope", "user", "--silent", "--disable-interactivity"}
	if !reflect.DeepEqual(f.calls[3], want) {
		t.Fatalf("install args=%v", f.calls[3])
	}
}

func TestNoApplicableInstallerExplainsUserScope(t *testing.T) {
	f := &fakeRunner{err: codeError(-1978335216)} // 0x8a150010
	err := New(f).Install(context.Background(), "Google.Chrome")
	if err == nil || !strings.Contains(err.Error(), "no applicable user-scope installer") || !strings.Contains(err.Error(), "Google.Chrome") {
		t.Fatalf("install error=%v", err)
	}
	want := []string{"install", "--id", "Google.Chrome", "--exact", "--source", "winget", "--scope", "user", "--silent", "--disable-interactivity"}
	if !reflect.DeepEqual(f.calls[1], want) {
		t.Fatalf("install args=%v", f.calls[1])
	}
}

func TestChromeEXEInstalledUsesExactPackageID(t *testing.T) {
	f := &fakeRunner{output: []byte("Name  Id  Version  Source\nGoogle Chrome  Google.Chrome.EXE  152.0.0  winget\n")}
	installed, err := New(f).Installed(context.Background(), "Google.Chrome.EXE")
	if err != nil || !installed {
		t.Fatalf("installed=%v error=%v", installed, err)
	}
	want := []string{"list", "--id", "Google.Chrome.EXE", "--exact", "--source", "winget", "--disable-interactivity"}
	if !reflect.DeepEqual(f.calls[1], want) {
		t.Fatalf("list args=%v", f.calls[1])
	}
}

func TestMachineScopePassedToWinGetInstall(t *testing.T) {
	f := &fakeRunner{}
	c := New(f)
	if err := c.Install(context.Background(), "wez.wezterm", "machine"); err != nil {
		t.Fatalf("install error: %v", err)
	}
	want := []string{"install", "--id", "wez.wezterm", "--exact", "--source", "winget", "--scope", "machine", "--silent", "--disable-interactivity"}
	if !reflect.DeepEqual(f.calls[1], want) {
		t.Fatalf("calls[1] = %v, want %v", f.calls[1], want)
	}
}

func TestAutoScopeJobOmitsScopeFilter(t *testing.T) {
	f := &fakeRunner{}
	job := New(f).Job("wez.wezterm", "WezTerm", "auto")
	if err := job.Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := []string{"install", "--id", "wez.wezterm", "--exact", "--source", "winget", "--silent", "--disable-interactivity"}
	if !reflect.DeepEqual(f.calls[1], want) {
		t.Fatalf("args=%v, want=%v", f.calls[1], want)
	}
}

func TestAutoScopeFailureDoesNotClaimMachineScope(t *testing.T) {
	f := &fakeRunner{err: codeError(-1978335216), output: []byte("installer diagnostics")}
	err := New(f).Install(context.Background(), "wez.wezterm", "auto")
	if err == nil || !strings.Contains(err.Error(), "without a scope filter") || !strings.Contains(err.Error(), "installer diagnostics") {
		t.Fatalf("error=%v", err)
	}
	if len(f.calls) != 2 {
		t.Fatalf("unexpected retry: %v", f.calls)
	}
}

func TestInvalidScopeDoesNotRun(t *testing.T) {
	f := &fakeRunner{}
	if err := New(f).Install(context.Background(), "Git.Git", "invalid"); err == nil {
		t.Fatal("invalid scope accepted")
	}
	if len(f.calls) != 0 {
		t.Fatalf("invalid scope ran WinGet: %v", f.calls)
	}
}

func TestInstallerCancellationGivesUACHintAndDoesNotRetry(t *testing.T) {
	f := &fakeRunner{err: codeError(-1978334964), output: []byte("The installer will request to run as administrator. Expect a prompt. You cancelled the installation. Installer failed with exit code: 2")}
	err := New(f).Install(context.Background(), "Git.Git")
	if err == nil || !strings.Contains(err.Error(), "UAC") || !strings.Contains(err.Error(), "exit code: 2") {
		t.Fatalf("error=%v", err)
	}
	if len(f.calls) != 2 {
		t.Fatalf("unexpected automatic retry: %v", f.calls)
	}
	f.output = []byte("Installer failed with exit code: 1603")
	err = New(f).Install(context.Background(), "Git.Git")
	if strings.Contains(err.Error(), "UAC") {
		t.Fatalf("generic installer failure misclassified: %v", err)
	}
}

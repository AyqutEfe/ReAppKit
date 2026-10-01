package winget

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

type promptRunner struct {
	fakeRunner
	prompts   [][]string
	answer    string
	promptErr error
}

func (r *promptRunner) RunInteractive(_ context.Context, input io.Reader, output io.Writer, args ...string) ([]byte, error) {
	r.prompts = append(r.prompts, append([]string(nil), args...))
	answer, _ := io.ReadAll(input)
	r.answer = string(answer)
	fmtOutput := []byte("Package agreement: [Y] Yes [N] No\n")
	output.Write(fmtOutput)
	return fmtOutput, r.promptErr
}

func TestPackageAgreementPromptUsesOriginalTerminalAndNoAutomaticAcceptance(t *testing.T) {
	for _, reject := range []bool{false, true} {
		r := &promptRunner{fakeRunner: fakeRunner{err: codeError(-1978335167)}} // 0x8a150041
		answer := "y\n"
		if reject {
			answer = "n\n"
			r.promptErr = codeError(-1978335167)
		}
		var output strings.Builder
		base := New(r).WithInteraction(strings.NewReader(answer), &output)
		err := base.WithSource("msstore").Install(context.Background(), "9NKSQGP7F2NH")
		if (err != nil) != reject || r.answer != answer || len(r.prompts) != 1 {
			t.Fatalf("reject=%v answer=%q prompts=%v err=%v", reject, r.answer, r.prompts, err)
		}
		want := []string{"install", "--id", "9NKSQGP7F2NH", "--exact", "--source", "msstore", "--silent"}
		if !reflect.DeepEqual(r.prompts[0], want) || len(r.calls) != 2 {
			t.Fatalf("interactive=%v silent calls=%v", r.prompts, r.calls)
		}
		if !strings.Contains(output.String(), "Package agreement") || base.sourceName() != "winget" {
			t.Fatalf("output=%s source=%s", output.String(), base.sourceName())
		}
		if reject && (!strings.Contains(err.Error(), "winget install --id 9NKSQGP7F2NH --exact --source msstore") || strings.Contains(err.Error(), "UAC")) {
			t.Fatalf("refusal guidance=%v", err)
		}
	}
}

func TestPackageAgreementWithoutTerminalGivesExactGuidance(t *testing.T) {
	r := &promptRunner{fakeRunner: fakeRunner{err: codeError(-1978335167)}}
	err := New(r).WithSource("msstore").Install(context.Background(), "9NT1R1C2HH7J")
	if err == nil || !strings.Contains(err.Error(), "winget install --id 9NT1R1C2HH7J --exact --source msstore") || len(r.prompts) != 0 {
		t.Fatalf("error=%v prompts=%v", err, r.prompts)
	}
}

type cacheRunner struct {
	fakeRunner
	queries    int
	persistent bool
}

func (r *cacheRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	if args[0] == "list" {
		r.queries++
		if r.queries == 1 || r.persistent {
			return nil, codeError(-int64(0x100000000 - 0x80071130))
		}
	}
	return r.fakeRunner.Run(ctx, args...)
}

func TestTransientCacheFailureRetriesReadOnlyQueryOnce(t *testing.T) {
	for _, persistent := range []bool{false, true} {
		r := &cacheRunner{fakeRunner: fakeRunner{output: []byte("MarkText MarkText.MarkText 0.19.1 winget\n")}, persistent: persistent}
		installed, err := New(r).Installed(context.Background(), "MarkText.MarkText")
		if r.queries != 2 || installed == persistent || (err != nil) != persistent {
			t.Fatalf("persistent=%v installed=%v queries=%d err=%v", persistent, installed, r.queries, err)
		}
		if persistent && !strings.Contains(err.Error(), "winget source update --name winget") {
			t.Fatalf("cache guidance=%v", err)
		}
	}
}

func TestHashMismatchIsNotUACCancellationAndDoesNotRetry(t *testing.T) {
	r := &promptRunner{fakeRunner: fakeRunner{err: codeError(-int64(0x100000000 - 0x8A150011)), output: []byte("Installer hash does not match")}}
	var output strings.Builder
	err := New(r).WithInteraction(strings.NewReader("y\n"), &output).Install(context.Background(), "Test.App")
	if err == nil || strings.Contains(err.Error(), "UAC") || len(r.calls) != 2 || len(r.prompts) != 0 {
		t.Fatalf("error=%v calls=%v prompts=%v", err, r.calls, r.prompts)
	}
}

type fakeRunner struct {
	calls  [][]string
	output []byte
	err    error
}

type sourcePromptRunner struct {
	fakeRunner
	accepted       bool
	refuse         bool
	persistent     bool
	prompts        int
	promptArgs     []string
	promptDeadline bool
	queryDeadlines []bool
}

func (r *sourcePromptRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	if args[0] == "list" {
		_, deadline := ctx.Deadline()
		r.queryDeadlines = append(r.queryDeadlines, deadline)
		if !r.accepted {
			return nil, codeError(-1978335162)
		}
	}
	return r.fakeRunner.Run(ctx, args...)
}

func (r *sourcePromptRunner) RunInteractive(ctx context.Context, input io.Reader, output io.Writer, args ...string) ([]byte, error) {
	r.prompts++
	r.promptArgs = append([]string(nil), args...)
	_, r.promptDeadline = ctx.Deadline()
	output.Write([]byte("Source terms: [Y] Yes [N] No\n"))
	if r.refuse {
		return nil, codeError(-1978335162)
	}
	if !r.persistent {
		r.accepted = true
	}
	return nil, codeError(-1978335212) // No installed package after acceptance.
}

func TestSourceReviewSeparatesUserPromptFromQueryDeadlineAndRechecks(t *testing.T) {
	for _, installed := range []bool{false, true} {
		r := &sourcePromptRunner{}
		if installed {
			r.output = []byte("WhatsApp 9NKSQGP7F2NH 1 msstore\n")
		}
		var output strings.Builder
		client := New(r).WithInteraction(strings.NewReader("y\n"), &output).WithSource("msstore")
		done, err := client.InstalledWithReview(context.Background(), "9NKSQGP7F2NH")
		want := []string{"list", "--id", "9NKSQGP7F2NH", "--exact", "--source", "msstore"}
		if err != nil || done != installed || r.prompts != 1 || !reflect.DeepEqual(r.promptArgs, want) || r.promptDeadline || !reflect.DeepEqual(r.queryDeadlines, []bool{true, true}) {
			t.Fatalf("installed=%v done=%v error=%v runner=%+v", installed, done, err, r)
		}
		if !strings.Contains(output.String(), "Source terms") {
			t.Fatalf("terms not shown: %s", output.String())
		}
	}
}

func TestSourceReviewRefusalOrUnpersistedAcceptanceNeverLoops(t *testing.T) {
	for _, persistent := range []bool{false, true} {
		r := &sourcePromptRunner{refuse: !persistent, persistent: persistent}
		var output strings.Builder
		client := New(r).WithInteraction(strings.NewReader("n\n"), &output).WithSource("msstore")
		done, err := client.InstalledWithReview(context.Background(), "9NKSQGP7F2NH")
		if done || err == nil || r.prompts != 1 || !strings.Contains(err.Error(), "winget list --id 9NKSQGP7F2NH --exact --source msstore") {
			t.Fatalf("done=%v error=%v prompts=%d", done, err, r.prompts)
		}
	}
}

func TestSourceReviewWithoutTerminalKeepsManualGuidance(t *testing.T) {
	r := &sourcePromptRunner{}
	done, err := New(r).WithSource("msstore").InstalledWithReview(context.Background(), "9NKSQGP7F2NH")
	if done || err == nil || r.prompts != 0 || !strings.Contains(err.Error(), "winget list --id 9NKSQGP7F2NH --exact --source msstore") {
		t.Fatalf("done=%v error=%v prompts=%d", done, err, r.prompts)
	}
}

func TestMicrosoftStoreSourceIsolationAndNoScopeOrAutomaticAgreements(t *testing.T) {
	f := &fakeRunner{output: []byte("ChatGPT 9NT1R1C2HH7J 1 msstore\n")}
	base := New(f)
	store := base.WithSource("msstore")
	installed, err := store.Installed(context.Background(), "9NT1R1C2HH7J")
	if err != nil || !installed {
		t.Fatalf("installed=%v error=%v", installed, err)
	}
	if f.calls[1][5] != "msstore" {
		t.Fatalf("Store query=%v", f.calls[1])
	}
	if err := store.Install(context.Background(), "9NT1R1C2HH7J"); err != nil {
		t.Fatal(err)
	}
	want := []string{"install", "--id", "9NT1R1C2HH7J", "--exact", "--source", "msstore", "--silent", "--disable-interactivity"}
	if !reflect.DeepEqual(f.calls[3], want) {
		t.Fatalf("Store install=%v", f.calls[3])
	}
	if base.sourceName() != "winget" {
		t.Fatal("Store client changed base source")
	}
	f.err = codeError(-1978335162)
	_, err = store.Installed(context.Background(), "9NT1R1C2HH7J")
	if err == nil || !strings.Contains(err.Error(), "--source msstore") {
		t.Fatalf("agreement guidance=%v", err)
	}
}

func TestInvalidSourceAndStoreMachineScopeNeverRun(t *testing.T) {
	f := &fakeRunner{}
	if err := New(f).WithSource("msstore").Install(context.Background(), "9NT1R1C2HH7J", "machine"); err == nil {
		t.Fatal("Store machine scope accepted")
	}
	if _, err := New(f).WithSource("unknown").Installed(context.Background(), "Test.App"); err == nil {
		t.Fatal("unknown source accepted")
	}
	if len(f.calls) != 0 {
		t.Fatalf("invalid request ran: %v", f.calls)
	}
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
	if err := c.Install(context.Background(), "Git.Git"); err == nil || !strings.Contains(err.Error(), "winget list --id Git.Git --exact --source winget") {
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

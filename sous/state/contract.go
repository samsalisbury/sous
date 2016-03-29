package core

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/opentable/sous/tools/cli"
	"github.com/opentable/sous/tools/cmd"
)

type Contracts map[string]Contract

type OrderedContracts []Contract

func (cs Contracts) Validate() error {
	for _, c := range cs {
		if err := c.Validate(); err != nil {
			return fmt.Errorf("Contract invalid: %s", err)
		}
	}
	return nil
}

func (cs Contracts) Clone() Contracts {
	contracts := make(Contracts, len(cs))
	for i, c := range cs {
		contracts[i] = c.Clone()
	}
	return contracts
}

type Contract struct {
	Name, Filename        string
	StartServers          List
	Values                Values
	Servers               map[string]TestServer
	Preconditions, Checks Checks
	SelfTest              ContractTest
}

type Values map[string]string

type List []string

func (l List) Clone() List {
	list := make([]string, len(l))
	copy(list, l)
	return list
}

func (vs Values) Clone() Values {
	values := make(map[string]string, len(vs))
	for k, v := range vs {
		values[k] = v
	}
	return values
}

func (c Contract) Clone() Contract {
	c.StartServers = c.StartServers.Clone()
	c.Values = c.Values.Clone()
	servers := make(map[string]TestServer)
	for k, v := range c.Servers {
		servers[k] = v
	}
	c.Servers = servers
	c.Preconditions = c.Preconditions.Clone()
	c.Checks = c.Checks.Clone()
	return c
}

type ContractTest struct {
	ContractName string
	CheckTests   []CheckTest
}

type CheckTest struct {
	CheckName  string
	TestImages struct {
		Pass, Fail string
	}
}

func (c Contract) Errorf(format string, a ...interface{}) error {
	f := c.Filename + ": " + format
	return fmt.Errorf(f, a...)
}

func (c Contract) Validate() error {
	if c.Name == "" {
		return c.Errorf("%s: Contract Name must not be empty")
	}
	if err := c.Preconditions.Validate(); err != nil {
		return c.Errorf("Precondition invalid: %s", err)
	}
	if err := c.Checks.Validate(); err != nil {
		return c.Errorf("Check invalid: %s", err)
	}
	return nil
}

type Checks []Check

func (cs Checks) Validate() error {
	for _, c := range cs {
		if err := c.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (cs Checks) Clone() Checks {
	checks := make([]Check, len(cs))
	copy(checks, cs)
	return checks
}

type TestServer struct {
	Name          string
	DefaultValues Values
	Startup       *StartupInfo
	Docker        DockerServer
}

func (ts TestServer) Clone() TestServer {
	ts.DefaultValues = ts.DefaultValues.Clone()
	ts.Startup = ts.Startup.Clone()
	ts.Docker = ts.Docker.Clone()
	return ts
}

type StartupInfo struct {
	CompleteWhen *Check
}

func (si StartupInfo) Clone() *StartupInfo {
	check := si.CompleteWhen.Clone()
	return &StartupInfo{
		&check,
	}
}

type DockerServer struct {
	Image         string
	Env           Values
	Options, Args List
}

func (ds DockerServer) Clone() DockerServer {
	ds.Env = ds.Env.Clone()
	ds.Options = ds.Options.Clone()
	ds.Args = ds.Args.Clone()
	return ds
}

type GetHTTPAssertion struct {
	URL, ResponseBodyContains, ResponseJSONContains string
	ResponseStatusCode                              int
	AnyResponse                                     bool
}

// Check MUST specify exactly one of GET, Shell, or Contract. If
// more than one of those are specified the check is invalid. This
// slightly ugly switching makes the YAML contract definitions
// much more readable, and is easily verifiable.
type Check struct {
	Name       string
	Timeout    time.Duration
	Setup      Action
	HTTPCheck  `yaml:",inline"`
	ShellCheck `yaml:",inline"`
}

func (ch Check) Clone() Check {
	return ch
}

type Action struct {
	Shell string
}

// Validate checks that we have a well-formed check.
func (c *Check) Validate() error {
	httpError := c.HTTPCheck.Validate()
	shellError := c.ShellCheck.Validate()
	if httpError != nil && shellError != nil {
		if c.HTTPCheck.GET != "" {
			return fmt.Errorf("%s", httpError)
		}
		if c.ShellCheck.Shell != "" {
			return fmt.Errorf("%s", shellError)
		}
		return fmt.Errorf("multiple errors: (%s) and (%s)", httpError, shellError)
	}
	if httpError == nil && shellError == nil {
		return fmt.Errorf("You have specified both Shell and GET, pick one or the other")
	}
	return nil
}

func (c *Check) Execute() error {
	if c.Setup.Shell != "" {
		cli.Verbosef("Running setup command...")
		cli.Verbosef("shell> %s", c.Setup.Shell)
		if code := wrapShellCommand(c.Setup.Shell).ExitCode(); code != 0 {
			return fmt.Errorf("Setup command failed: exit code %d", code)
		}
	}
	if c.HTTPCheck.Validate() == nil {
		return Within(c.Timeout, "", false, func() error {
			return c.HTTPCheck.Execute()
		})
	}
	if c.ShellCheck.Validate() == nil {
		return Within(c.Timeout, "", false, func() error {
			return c.ShellCheck.Execute()
		})
	}
	return c.Validate()
}

type HTTPCheck struct {
	// GET must be a URL, or empty if Shell is not empty.
	// The following 4 fields are assertions about
	// the response after getting that URL via HTTP.
	GET                      string
	StatusCode               int
	StatusCodeRange          []int
	BodyContainsString       string
	BodyDoesNotContainString string
}

// Validate HTTPCheck, return an error if it is not valid.
func (c HTTPCheck) Validate() error {
	if c.GET == "" {
		return fmt.Errorf("GET not specified")
	}
	if c.StatusCode == 0 && len(c.StatusCodeRange) == 0 && c.BodyContainsString == "" && c.BodyDoesNotContainString == "" {
		return fmt.Errorf("you must supply at least one of: StatusCode, StatusCodeRange, BodyContainsString, BodyDoesNotContainString")
	}
	if c.StatusCode < 0 || c.StatusCode > 999 {
		return fmt.Errorf("StatusCode was %d; want 0 ≤ StatusCode ≤ 999")
	}
	if len(c.StatusCodeRange) == 1 || len(c.StatusCodeRange) > 2 {
		return fmt.Errorf("StatusCodeRange was %v; want it to be empty or contain exactly 2 elements")
	}
	return nil
}

// Execute an HTTPCheck, you must first check it is valid with Validate, or the behaviour is undefined.
func (c HTTPCheck) Execute() error {
	response, err := http.Get(c.GET)
	if err != nil {
		return err
	}
	if response.Body != nil {
		defer response.Body.Close()
	}
	if c.StatusCode != 0 && response.StatusCode != c.StatusCode {
		return fmt.Errorf("got status code %d; want %d", response.StatusCode, c.StatusCode)
	}
	if len(c.StatusCodeRange) != 0 {
		if response.StatusCode < c.StatusCodeRange[0] || response.StatusCode > c.StatusCodeRange[1] {
			return fmt.Errorf("got status code %s; want something in the range %d..%d", response.StatusCode, c.StatusCodeRange[0], c.StatusCodeRange[1])
		}
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("unable to read response body: %s", err)
	}
	if c.BodyContainsString != "" && !strings.Contains(string(body), c.BodyContainsString) {
		return fmt.Errorf("expected to find string %q in body but did not", c.BodyContainsString)
	}
	if c.BodyDoesNotContainString != "" && strings.Contains(string(body), c.BodyDoesNotContainString) {
		return fmt.Errorf("found string %q in body, expected not to", c.BodyDoesNotContainString)
	}
	return nil
}

type ShellCheck struct {
	// Shell must be a valid POSIX shell command, or empty if GET is not
	// empty. The command will be executed and the exit code checked
	// against the expected code (note that ints default to zero, so the
	// default case is that we expect a success (0) exit code.
	Shell    string
	ExitCode int
}

func (c ShellCheck) Validate() error {
	if c.Shell == "" {
		return fmt.Errorf("Shell command not specified")
	}
	return nil
}

// Wrap the command in a subshell so the command can contain pipelines.
// Note that the spaces between the parentheses are mandatory for compatibility
// with further subshells defined in the contract, so don't remove them.
func wrapShellCommand(command string) *cmd.CMD {
	return cmd.New("/bin/sh", "-c", fmt.Sprintf("( %s )", command))
}

func (c ShellCheck) Execute() error {
	if code := wrapShellCommand(c.Shell).ExitCode(); code != c.ExitCode {
		return fmt.Errorf("got exit code %d; want %d", code, c.ExitCode)
	}
	return nil
}

func (c Check) String() string {
	if c.Name != "" {
		return c.Name
	}
	if c.Shell != "" {
		return c.Shell
	}
	if c.GET != "" {
		return fmt.Sprintf("GET %s", c.GET)
	}
	return "INVALID CHECK"
}
func Within(d time.Duration, action string, showProgress bool, f func() error) error {
	start := time.Now()
	end := start.Add(d)
	tryCount := 0
	var p cli.Progress
	for {
		tryCount++
		err := f()
		if err == nil {
			p.Done("Success")
			return nil
		}
		if time.Now().After(end) {
			p.Done("Timeout")
			return err
		}
		// Don't show progress until we've tried a hundred times.
		if tryCount > 100 && p == "" && showProgress {
			p = cli.BeginProgress(action)
		}
		if tryCount%100 == 0 {
			p.Increment()
		}
		time.Sleep(10 * time.Millisecond)
	}
}

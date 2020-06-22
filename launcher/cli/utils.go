package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/lithammer/dedent"
	"github.com/logrusorgru/aurora"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func mustReplaceDataDir(dataDir, in string) string {
	d, err := filepath.Abs(dataDir)
	if err != nil {
		panic(fmt.Errorf("file path abs: %w", err))
	}

	in = strings.Replace(in, "{dfuse-data-dir}", d, -1)
	return in
}

func mkdirStorePathIfLocal(storeURL string) (err error) {
	userLog.Debug("creating directory and its parent(s)", zap.String("directory", storeURL))
	if dirs := getDirsToMake(storeURL); len(dirs) > 0 {
		err = makeDirs(dirs)
	}
	return
}

func getDirsToMake(storeURL string) []string {
	parts := strings.Split(storeURL, "://")
	if len(parts) > 1 {
		if parts[0] != "file" {
			// Not a local store, nothing to do
			return nil
		}
		storeURL = parts[1]
	}

	// Some of the store URL are actually a file directly, let's try our best to cope for that case
	filename := filepath.Base(storeURL)
	if strings.Contains(filename, ".") {
		storeURL = filepath.Dir(storeURL)
	}

	// If we reach here, it's a local store path
	return []string{storeURL}
}

var deepMindVersionRegexp = regexp.MustCompile("dm[-\\.]([1-9][0-9]*)\\.([0-9]+)")
var nodeosVersionRegexp = regexp.MustCompile("v?([0-9]+)\\.([0-9]+)\\.([0-9]+)(-(.*))?")

type nodeosVersion struct {
	full string

	major  int
	minor  int
	patch  int
	suffix string
}

// NewNodeosVersionFromSystem runs the `nodeos` binary found in `PATH` enviornment
// variable and extract the version from it.
func newNodeosVersionFromSystem() (out nodeosVersion, err error) {
	cmd := exec.Command(viper.GetString("global-nodeos-path"), "--version")
	stdOut, err := cmd.Output()
	if err != nil {
		err = fmt.Errorf("unable to run command %q: %w", cmd.String(), err)
		return
	}

	return newNodeosVersionFromString(string(stdOut))
}

// NewNodeosVersionFromString parsed the received string and return a structured object
// representing the version information.
func newNodeosVersionFromString(version string) (out nodeosVersion, err error) {
	matches := nodeosVersionRegexp.FindAllStringSubmatch(version, -1)
	if len(matches) == 0 {
		err = fmt.Errorf("unable to parse version %q, expected to match %s", version, nodeosVersionRegexp)
		return
	}

	userLog.Debug("nodeos version regexp matched", zap.Reflect("matches", matches))

	// We don't care for multiple matches for now
	match := matches[0]
	out.full = match[0]

	// We skip the errors since the regex match only digits on those groups
	out.major, _ = strconv.Atoi(match[1])
	out.minor, _ = strconv.Atoi(match[2])
	out.patch, _ = strconv.Atoi(match[3])

	if len(match) >= 5 {
		out.suffix = match[5]
	}

	return
}

func (v nodeosVersion) String() string {
	return v.full
}

func (v nodeosVersion) supportsDeepMind(deepMindMajor int) bool {
	// FIXME: This check is good only for releases prepared by dfuse Team directly.
	//        When we are going to use the upstream version of EOSIO, this is not going
	//        to work as expected! At the same time, there is nothing else that can be
	//        done just now, we could check that he version is above a certain value.
	if !strings.Contains(v.suffix, "dm") {
		return false
	}

	matches := deepMindVersionRegexp.FindAllStringSubmatch(v.suffix, -1)
	if len(matches) != 1 {
		userLog.Debug("unable to parse deep mind version", zap.String("deep_mind_version", v.suffix), zap.Stringer("regexp", deepMindVersionRegexp))
		return false
	}

	match := matches[0]

	// We skip the errors since the regex match only digits on those groups
	major, _ := strconv.Atoi(match[1])

	return major == deepMindMajor
}

func maybeCheckNodeosVersion() {
	if viper.GetBool("skip-checks") {
		return
	}

	version, err := newNodeosVersionFromSystem()
	if err != nil {
		userLog.Debug("unable to extract nodeos version from system", zap.Error(err))
		cliErrorAndExit(dedentf(`
			We were unable to detect "nodeos" version on your system. This can be due to
			one of the following reasons:
			- You don't have "nodeos" installed on your system
			- It's installed but not referred by your PATH environment variable, so we did not find it
			- It's installed but execution of "nodeos --version" failed

			Make sure you have a dfuse instrumented 'nodeos' binary, follow instructions
			at https://github.com/dfuse-io/dfuse-eosio/blob/develop/DEPENDENCIES.md#dfuse-instrumented-eosio-prebuilt-binaries
			to find how to install it.

			If you have your dfuse instrumented 'nodeos' binary outside your PATH, use --nodeos-path=<location>
			argument to specify path to it.

			If you think this is a mistake, you can re-run this command adding --skip-checks, which
			will not perform this check.
		`))
	}

	if !version.supportsDeepMind(12) {
		// Upgrade message for those already using a deep mind aware `nodeos` but not
		// of the correct major version.
		if strings.Contains(version.String(), "dm") {
			cliErrorAndExit(dedentf(`
				The "nodeos" binary found on your system with version %s is not supported by this
				version of dfuse for EOSIO. We recently made incompatible changes to the deep mind
				code found in "nodeos" binary that requires you to upgrade it.

				Follow instructions at https://github.com/dfuse-io/dfuse-eosio/blob/develop/DEPENDENCIES.md#dfuse-instrumented-eosio-prebuilt-binaries
				to find the latest version to install for your platform.

				If you think this is a mistake, you can re-run this command adding --skip-checks, which
				will not perform this check.
			`, version))
		}

		cliErrorAndExit(dedentf(`
			The "nodeos" binary found on your system with version %s does not seem to be a dfuse
			instrumented binary. Maybe your dfuse instrumented 'nodeos' binary is not in your
			PATH environment variable?

			Make sure you have a dfuse instrumented 'nodeos' binary, follow instructions
			at https://github.com/dfuse-io/dfuse-eosio/blob/develop/DEPENDENCIES.md#dfuse-instrumented-eosio-prebuilt-binaries
			to find how to install it.

			If you have your dfuse instrumented 'nodeos' binary outside your PATH, use --nodeos-path=<location>
			argument to specify path to it.

			If you think this is a mistake, you can re-run this command adding --skip-checks, which
			will not perform this check.
		`, version))
	}
}

func cliErrorAndExit(message string, args ...interface{}) {
	fmt.Println(aurora.Red(fmt.Sprintf(message, args...)).String())
	os.Exit(1)
}

func dedentf(format string, args ...interface{}) string {
	return fmt.Sprintf(dedent.Dedent(strings.TrimPrefix(format, "\n")), args...)
}

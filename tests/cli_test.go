package tests

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/acarl005/stripansi"
	dashboard "github.com/dfuse-io/dfuse-eosio/dashboard/pb"
	"github.com/dfuse-io/dfuse-eosio/launcher/cli"
	"github.com/dfuse-io/dgrpc"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var zlog = zap.NewNop()

func init() {
	if os.Getenv("DEBUG") != "" {
		logger, _ := zap.NewDevelopment()
		zlog = logger
	}
}

func TestCli(t *testing.T) {
	if os.Getenv("E2E_TESTS") != "true" {
		t.Skip("You must set environment variable 'E2E_TESTS=true' to run this test for now")
	}

	// FIXME: Need to find a way to ensure the binary is built and at the latest version, maybe
	//        we could invoke from the test directly the `go build -o somewhere ./cmd/dfuseeos`
	//        ourself?
	binaryPath, err := filepath.Abs("../dfuseeos")
	require.NoError(t, err)

	dataDir, cleanup := setupExecution(t, "happy-path", "./testdata/start-producer-config.yaml")
	defer cleanup()

	fmt.Printf("Test Config [Binary %q, Working Dir %q]\n", binaryPath, dataDir)

	ctx, cancelTimeout := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelTimeout()

	cmd := NewCommand(ctx, binaryPath, "start")
	cmd.cmd.Dir = dataDir
	cmd.Start()
	if cmd.Error() != nil {
		reportCommandFailure(t, cmd)
		return
	}

	notReadyApps := waitForAllAppsToBeReady(t, 50*time.Second)
	require.Empty(t, notReadyApps)

	// Make other tests

	cmd.SendInterrupt()
	cmd.Wait()

	assertSuccess(t, cmd)
	assertStdoutContains(t, cmd, "Dashboard: http://localhost:8080")
	assertStdoutContains(t, cmd, "GraphiQL: http://localhost:8080/graphiql")
	assertStdoutContains(t, cmd, "Eosq: http://localhost:8081")
	assertStdoutContains(t, cmd, "Received termination signal, quitting")
	assertStdoutContains(t, cmd, "Goodbye")

	if os.Getenv("DEBUG") != "" {
		stdOut, stdErr := cmdOutputs(cmd)
		fmt.Printf("\nCommand outputs\nStderr\n%s\n\nStdout\n%s\n", stdErr, stdOut)
	}
}

func waitForAllAppsToBeReady(t *testing.T, timeout time.Duration) (notReadyApps []string) {
	conn, err := dgrpc.NewInternalClient("localhost" + cli.DashboardGrpcServingAddr)
	require.NoError(t, err)

	// FIXME: Connect this context here to the test context so that if the test process is killed,
	//        this call would stop by itself.
	ctx := context.Background()
	client := dashboard.NewDashboardClient(conn)

	readyMap := collectApps(t, client)

	streamCtx, cancelTimeout := context.WithTimeout(ctx, timeout)
	defer cancelTimeout()

	stream, err := client.AppsInfo(streamCtx, &dashboard.AppsInfoRequest{})
	require.NoError(t, err)

	for {
		message, err := stream.Recv()
		if err == io.EOF {
			zlog.Debug("received EOF on app status stream")
			return collectNotReadyApps(readyMap)
		}

		code := status.Convert(err).Code()
		if code != codes.OK {
			zlog.Debug("did not complete correctly", zap.Stringer("code", code))
			return collectNotReadyApps(readyMap)
		}

		for _, appInfo := range message.Apps {
			zlog.Debug("setting readiness of app", zap.String("app_id", appInfo.Id), zap.Bool("ready", appInfo.Status == dashboard.AppStatus_RUNNING))
			readyMap[appInfo.Id] = appInfo.Status == dashboard.AppStatus_RUNNING
		}

		if allAppsReady(readyMap) {
			zlog.Debug("all apps ready, terminating wait")
			return nil
		}
	}
}

func allAppsReady(readyMap map[string]bool) bool {
	for _, isReady := range readyMap {
		if !isReady {
			return false
		}
	}

	return true
}

func collectApps(t *testing.T, client dashboard.DashboardClient) (readyMap map[string]bool) {
	appListResponse, err := client.AppsList(context.Background(), &dashboard.AppsListRequest{})
	require.NoError(t, err)

	readyMap = map[string]bool{}
	for _, appInfo := range appListResponse.Apps {
		zlog.Debug("adding app to list of app to be ready", zap.String("app_id", appInfo.Id))
		readyMap[appInfo.Id] = false
	}

	return readyMap
}

func collectNotReadyApps(readyMap map[string]bool) (notReadyApps []string) {
	for appID, isReady := range readyMap {
		if !isReady {
			notReadyApps = append(notReadyApps, appID)
		}
	}

	return notReadyApps
}

func assertSuccess(t *testing.T, cmd *Command) {
	if !cmd.Success() {
		reportCommandFailure(t, cmd)
	}
}

func assertStdoutContains(t *testing.T, cmd *Command, str string) {
	stdOut, stdErr := cmdOutputs(cmd)
	if !cmd.StdoutContains(str) {
		require.Fail(t, fmt.Sprintf("Expecting stdout to contain %q but did not find anything matching", str), "Stderr\n%s\n\nStdout\n%s", stdErr, stdOut)
	}
}

func reportCommandFailure(t *testing.T, cmd *Command) {
	stdOut, stdErr := cmdOutputs(cmd)
	require.Fail(t, "Command failed", "%s\n\nStderr\n%s\n\nStdout\n%s", cmd.Error(), stdErr, stdOut)
}

func cmdOutputs(cmd *Command) (stdOut, stdErr string) {
	stdOut = stripansi.Strip(cmd.Stdout())
	if stdOut == "" {
		stdOut = "<Nothing in standard out>"
	}

	stdErr = stripansi.Strip(cmd.Stderr())
	if stdErr == "" {
		stdErr = "<Nothing in standard error>"
	}

	return
}

func setupExecution(t *testing.T, testCase string, configFile string) (dataDir string, cleanup func()) {
	var err error
	dataDir, err = ioutil.TempDir(os.TempDir(), testCase)
	require.NoError(t, err)

	err = os.MkdirAll(dataDir, os.ModePerm)
	require.NoError(t, err)

	content, err := ioutil.ReadFile(configFile)
	require.NoError(t, err)

	err = ioutil.WriteFile(filepath.Join(dataDir, "dfusebox.yaml"), content, os.ModePerm)
	require.NoError(t, err)

	return dataDir, func() {
		// Let's delete the actual temporary dir only if DEBUG env is not set,
		// so to debug a failing test, simply use `DEBUG=something` and the folder
		// will not be deleted.
		if os.Getenv("DEBUG") == "" {
			// Too bad for errors, we cannot do anything
			os.RemoveAll(dataDir)
		}
	}
}

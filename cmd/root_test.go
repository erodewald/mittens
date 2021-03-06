//Copyright 2019 Expedia, Inc.
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

// +build integration

package cmd

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"mittens/fixture"
	"mittens/pkg/probe"
	"mittens/pkg/safe"
	"net/http"
	"os"
	"testing"
)

var mockHttpServer *http.Server
var mockGrpcServer *grpc.Server

func TestMain(m *testing.M) {
	setup()
	m.Run()
	teardown()
}

func TestShouldBeReadyRegardlessIfWarmupRan(t *testing.T) {
	probe.DeleteFile("alive")
	probe.DeleteFile("ready")

	os.Args = []string{"mittens",
		"-file-probe-enabled=true",
		"-http-requests=get:/non-existent",
		"-concurrency=2",
		"-exit-after-warmup=true",
		"-target-readiness-http-path=/health",
		"-max-duration-seconds=5"}

	CreateConfig()
	RunCmdRoot()

	assert.Equal(t, true, opts.FileProbe.Enabled)
	assert.ElementsMatch(t, opts.HTTP.Requests, []string{"get:/non-existent"})
	assert.Equal(t, 2, opts.Concurrency)
	assert.Equal(t, true, opts.ExitAfterWarmup)
	assert.Equal(t, "/health", opts.Target.ReadinessHTTPPath)
	assert.Equal(t, 5, opts.MaxDurationSeconds)

	readyFileExists, err := probe.FileExists("ready")
	require.NoError(t, err)
	assert.True(t, readyFileExists)
}

func TestShouldBeReadyRegardlessIfHasPanicked(t *testing.T) {
	probe.DeleteFile("ready")

	// we trigger a panic scenario by using a non-existent gRPC readiness probe
	os.Args = []string{"mittens",
		"-file-probe-enabled=true",
		"-exit-after-warmup=true",
		"-fail-readiness=false",
		"-target-readiness-protocol=grpc",
		"-target-grpc-port=50051",
		"-target-readiness-grpc-method=non.existent/NonExistent",
		"-target-insecure=true",
		"-max-duration-seconds=5"}

	CreateConfig()
	RunCmdRoot()

	assert.True(t, safe.HasPanicked())
	readyFileExists, err := probe.FileExists("ready")
	require.NoError(t, err)
	assert.True(t, readyFileExists)
}

func TestWarmupSidecarWithFileProbe(t *testing.T) {
	probe.DeleteFile("alive")
	probe.DeleteFile("ready")

	os.Args = []string{"mittens",
		"-file-probe-enabled=true",
		"-http-requests=get:/delay",
		"-concurrency=2",
		"-exit-after-warmup=true",
		"-target-readiness-http-path=/health",
		"-max-duration-seconds=5"}

	CreateConfig()
	RunCmdRoot()

	assert.Equal(t, true, opts.FileProbe.Enabled)
	assert.ElementsMatch(t, opts.HTTP.Requests, []string{"get:/delay"})
	assert.Equal(t, 2, opts.Concurrency)
	assert.Equal(t, true, opts.ExitAfterWarmup)
	assert.Equal(t, "/health", opts.Target.ReadinessHTTPPath)
	assert.Equal(t, 5, opts.MaxDurationSeconds)

	readyFileExists, err := probe.FileExists("ready")
	require.NoError(t, err)
	assert.True(t, readyFileExists)
}

func TestWarmupFailReadinessIfTargetIsNeverReady(t *testing.T) {
	probe.DeleteFile("alive")
	probe.DeleteFile("ready")

	// we simulate a failure in the target by setting the readiness path to a non existent one so that
	// the target never becomes ready and the warmup does not run
	os.Args = []string{"mittens",
		"-file-probe-enabled=true",
		"-http-requests=get:/delay",
		"-target-readiness-port=8080",
		"-target-readiness-http-path=/non-existent",
		"-max-duration-seconds=5",
		"-exit-after-warmup=true",
		"-fail-readiness=true"}

	CreateConfig()
	RunCmdRoot()

	assert.Equal(t, true, opts.FileProbe.Enabled)
	assert.ElementsMatch(t, opts.HTTP.Requests, []string{"get:/delay"})
	assert.Equal(t, true, opts.ExitAfterWarmup)
	assert.Equal(t, "/non-existent", opts.Target.ReadinessHTTPPath)
	assert.Equal(t, 5, opts.MaxDurationSeconds)
	assert.Equal(t, true, opts.FailReadiness)

	readyFileExists, err := probe.FileExists("ready")
	require.NoError(t, err)
	assert.False(t, readyFileExists)
}

func TestWarmupFailReadinessIfNoRequestsAreSentToTarget(t *testing.T) {
	probe.DeleteFile("alive")
	probe.FileExists("ready")

	// we simulate a failure by using a port that doesnt exist (9999)
	os.Args = []string{"mittens",
		"-file-probe-enabled=true",
		"-http-requests=get:/delay",
		"-target-http-port=9999",
		"-target-readiness-port=8080",
		"-target-readiness-http-path=/health",
		"-max-duration-seconds=5",
		"-exit-after-warmup=true",
		"-fail-readiness=true"}

	CreateConfig()
	RunCmdRoot()

	assert.Equal(t, true, opts.FileProbe.Enabled)
	assert.ElementsMatch(t, opts.HTTP.Requests, []string{"get:/delay"})
	assert.Equal(t, true, opts.ExitAfterWarmup)
	assert.Equal(t, "/health", opts.Target.ReadinessHTTPPath)
	assert.Equal(t, 5, opts.MaxDurationSeconds)
	assert.Equal(t, true, opts.FailReadiness)

	readyFileExists, err := probe.FileExists("ready")
	require.NoError(t, err)
	assert.False(t, readyFileExists)
}

func TestGrpcAndHttp(t *testing.T) {
	probe.DeleteFile("alive")
	probe.DeleteFile("ready")

	os.Args = []string{"mittens",
		"-file-probe-enabled=true",
		"-target-grpc-port=50051",
		"-http-requests=get:/delay",
		"-grpc-requests=grpc.testing.TestService/EmptyCall",
		"-grpc-requests=grpc.testing.TestService/UnaryCall:{\"payload\":{\"body\":\"abcdefghijklmnopqrstuvwxyz01\"}}",
		"-target-insecure=true",
		"-concurrency=2",
		"-exit-after-warmup=true",
		"-target-readiness-http-path=/health",
		"-max-duration-seconds=5"}

	CreateConfig()
	RunCmdRoot()

	assert.Equal(t, true, opts.FileProbe.Enabled)
	assert.ElementsMatch(t, opts.HTTP.Requests, []string{"get:/delay"})
	assert.ElementsMatch(t, opts.Grpc.Requests, []string{"grpc.testing.TestService/EmptyCall", "grpc.testing.TestService/UnaryCall:{\"payload\":{\"body\":\"abcdefghijklmnopqrstuvwxyz01\"}}"})

	assert.Equal(t, 2, opts.Concurrency)
	assert.Equal(t, true, opts.ExitAfterWarmup)
	assert.Equal(t, "/health", opts.Target.ReadinessHTTPPath)
	assert.Equal(t, 5, opts.MaxDurationSeconds)

	readyFileExists, err := probe.FileExists("ready")
	require.NoError(t, err)
	assert.True(t, readyFileExists)
}

func setup() {
	mockHttpServer = fixture.StartHttpTargetTestServer(8080, []fixture.PathResponseHandler{})
	mockGrpcServer = fixture.StartGrpcTargetTestServer(50051)
}

func teardown() {
	mockHttpServer.Close()
	mockGrpcServer.Stop()
}

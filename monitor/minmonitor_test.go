package monitor

import (
	"bufio"
	"fmt"
	"github.com/fitstar/falcore"
	"github.com/proidiot/gone/errors"
	"github.com/stretchr/testify/assert"
	configutil "github.com/stuphlabs/pullcord/config/util"
	"github.com/stuphlabs/pullcord/util"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// serveLandingPage is a testing helper function that creates a webserver that
// other tests for MinSession can use to verify monitoring service.
func serveLandingPage(landingServer *falcore.Server) {
	err := landingServer.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

// TestMinMonitorUpService verifies that a MinMonitor generated by
// NewMinMonitor will give the expected status for a service that is up.
func TestMinMonitorUpService(t *testing.T) {
	testServiceName := "test"
	testHost := "localhost"
	testProtocol := "tcp"
	gracePeriod := time.Duration(0)

	landingPipeline := falcore.NewPipeline()
	landingPipeline.Upstream.PushBack(&util.LandingFilter{})
	landingServer := falcore.NewServer(0, landingPipeline)
	go serveLandingPage(landingServer)
	defer landingServer.StopAccepting()

	<- landingServer.AcceptReady

	service, err := NewMinMonitorredService(
		testHost,
		landingServer.Port(),
		testProtocol,
		gracePeriod,
		nil,
		nil,
		nil,
	)
	assert.NoError(t, err)
	mon := MinMonitor{}
	err = mon.Add(
		testServiceName,
		service,
	)
	assert.NoError(t, err)

	up, err := mon.Status(testServiceName)
	assert.NoError(t, err)
	assert.True(t, up)
}

// TestMinMonitorDownService verifies that a MinMonitor generated by
// NewMinMonitor will give the expected status for a service that is up.
func TestMinMonitorDownService(t *testing.T) {
	testServiceName := "test"
	testHost := "localhost"
	testProtocol := "tcp"
	gracePeriod := time.Duration(0)

	server, err := net.Listen(testProtocol, ":0")
	assert.NoError(t, err)
	_, rawPort, err := net.SplitHostPort(server.Addr().String())
	assert.NoError(t, err)
	testPort, err := strconv.Atoi(rawPort)
	assert.NoError(t, err)
	err = server.Close()
	assert.NoError(t, err)

	svc, err := NewMinMonitorredService(
		testHost,
		testPort,
		testProtocol,
		gracePeriod,
		nil,
		nil,
		nil,
	)
	assert.NoError(t, err)
	mon := MinMonitor{}
	err = mon.Add(
		testServiceName,
		svc,
	)
	assert.NoError(t, err)

	up, err := mon.Status(testServiceName)
	assert.NoError(t, err)
	assert.False(t, up)
}

// TestMinMonitorInvalidService verifies that a MinMonitor generated by
// NewMinMonitor will give the expected status for a service that is up.
func TestMinMonitorInvalidService(t *testing.T) {
	testServiceName := "test"
	testHost := "256.256.256.256.256"
	testPort := 80
	testProtocol := "tcp"
	gracePeriod := time.Duration(0)

	svc, err := NewMinMonitorredService(
		testHost,
		testPort,
		testProtocol,
		gracePeriod,
		nil,
		nil,
		nil,
	)
	assert.NoError(t, err)
	mon := MinMonitor{}
	err = mon.Add(
		testServiceName,
		svc,
	)
	assert.NoError(t, err)

	up, err := mon.Status(testServiceName)
	assert.Error(t, err)
	assert.False(t, up)
}

// TestMinMonitorUpReprobe verifies that a MinMonitor generated by
// NewMinMonitor will give the expected status for a service that is up.
func TestMinMonitorUpReprobe(t *testing.T) {
	testServiceName := "test"
	testHost := "localhost"
	testProtocol := "tcp"
	gracePeriod := time.Duration(0)

	landingPipeline := falcore.NewPipeline()
	landingPipeline.Upstream.PushBack(&util.LandingFilter{})
	landingServer := falcore.NewServer(0, landingPipeline)
	go serveLandingPage(landingServer)
	defer landingServer.StopAccepting()

	<- landingServer.AcceptReady

	svc, err := NewMinMonitorredService(
		testHost,
		landingServer.Port(),
		testProtocol,
		gracePeriod,
		nil,
		nil,
		nil,
	)
	assert.NoError(t, err)
	mon := MinMonitor{}
	err = mon.Add(
		testServiceName,
		svc,
	)
	assert.NoError(t, err)

	up, err := mon.Reprobe(testServiceName)
	assert.NoError(t, err)
	assert.True(t, up)
}

// TestMinMonitorDownReprobe verifies that a MinMonitor generated by
// NewMinMonitor will give the expected status for a service that is up.
func TestMinMonitorDownReprobe(t *testing.T) {
	testServiceName := "test"
	testHost := "localhost"
	testProtocol := "tcp"
	gracePeriod := time.Duration(0)

	server, err := net.Listen(testProtocol, ":0")
	assert.NoError(t, err)
	_, rawPort, err := net.SplitHostPort(server.Addr().String())
	assert.NoError(t, err)
	testPort, err := strconv.Atoi(rawPort)
	assert.NoError(t, err)
	err = server.Close()
	assert.NoError(t, err)

	svc, err := NewMinMonitorredService(
		testHost,
		testPort,
		testProtocol,
		gracePeriod,
		nil,
		nil,
		nil,
	)
	assert.NoError(t, err)
	mon := MinMonitor{}
	err = mon.Add(
		testServiceName,
		svc,
	)
	assert.NoError(t, err)

	up, err := mon.Reprobe(testServiceName)
	assert.NoError(t, err)
	assert.False(t, up)
}

// TestMinMonitorSetStatusUp verifies that a MinMonitor generated by
// NewMinMonitor will give the expected status for a service that is up.
func TestMinMonitorSetStatusUp(t *testing.T) {
	testServiceName := "test"
	testHost := "localhost"
	testProtocol := "tcp"
	gracePeriod, err := time.ParseDuration("30s")
	assert.NoError(t, err)

	server, err := net.Listen(testProtocol, ":0")
	assert.NoError(t, err)
	_, rawPort, err := net.SplitHostPort(server.Addr().String())
	assert.NoError(t, err)
	testPort, err := strconv.Atoi(rawPort)
	assert.NoError(t, err)
	err = server.Close()
	assert.NoError(t, err)

	svc, err := NewMinMonitorredService(
		testHost,
		testPort,
		testProtocol,
		gracePeriod,
		nil,
		nil,
		nil,
	)
	assert.NoError(t, err)
	mon := MinMonitor{}
	err = mon.Add(
		testServiceName,
		svc,
	)
	assert.NoError(t, err)

	up, err := mon.Status(testServiceName)
	assert.NoError(t, err)
	assert.False(t, up)

	mon.SetStatusUp(testServiceName)

	up, err = mon.Status(testServiceName)
	assert.NoError(t, err)
	assert.True(t, up)
}

// TestMinMonitorFalsePositive verifies that a MinMonitor generated by
// NewMinMonitor will give the expected status for a service that is up.
func TestMinMonitorFalsePositive(t *testing.T) {
	testServiceName := "test"
	testHost := "localhost"
	testProtocol := "tcp"
	gracePeriod, err := time.ParseDuration("30s")
	assert.NoError(t, err)

	landingPipeline := falcore.NewPipeline()
	landingPipeline.Upstream.PushBack(&util.LandingFilter{})
	landingServer := falcore.NewServer(0, landingPipeline)
	go serveLandingPage(landingServer)

	<- landingServer.AcceptReady

	service, err := NewMinMonitorredService(
		testHost,
		landingServer.Port(),
		testProtocol,
		gracePeriod,
		nil,
		nil,
		nil,
	)
	assert.NoError(t, err)
	mon := MinMonitor{}
	err = mon.Add(
		testServiceName,
		service,
	)
	assert.NoError(t, err)

	up, err := mon.Status(testServiceName)
	assert.NoError(t, err)
	assert.True(t, up)

	landingServer.StopAccepting()

	up, err = mon.Status(testServiceName)
	assert.NoError(t, err)
	assert.True(t, up)
}

// TestMinMonitorTrueNegative verifies that a MinMonitor generated by
// NewMinMonitor will give the expected status for a service that is up.
func TestMinMonitorTrueNegative(t *testing.T) {
	testServiceName := "test"
	testHost := "localhost"
	testProtocol := "tcp"
	gracePeriod := time.Duration(0)

	server, err := net.Listen(testProtocol, ":0")
	assert.NoError(t, err)
	_, rawPort, err := net.SplitHostPort(server.Addr().String())
	assert.NoError(t, err)
	testPort, err := strconv.Atoi(rawPort)
	assert.NoError(t, err)

	service, err := NewMinMonitorredService(
		testHost,
		testPort,
		testProtocol,
		gracePeriod,
		nil,
		nil,
		nil,
	)
	assert.NoError(t, err)
	mon := MinMonitor{}
	err = mon.Add(
		testServiceName,
		service,
	)
	assert.NoError(t, err)

	up, err := mon.Status(testServiceName)
	assert.NoError(t, err)
	assert.True(t, up)

	err = server.Close()
	assert.NoError(t, err)

	// The socket is kept open for an amount of time after being prompted
	// to close in case any more TCP packets show up. Unfortunately we'll
	// just have to wait.
	tcpTimeoutFile, err := os.Open(
		"/proc/sys/net/ipv4/tcp_fin_timeout",
	)
	defer tcpTimeoutFile.Close()
	assert.NoError(t, err)
	tcpTimeoutReader := bufio.NewReader(tcpTimeoutFile)
	line, err := tcpTimeoutReader.ReadString('\n')
	assert.NoError(t, err)
	line = line[:len(line) - 1]
	tcpTimeout, err := strconv.Atoi(line)
	assert.NoError(t, err)
	sleepSeconds := tcpTimeout + 1
	sleepDuration, err := time.ParseDuration(
		fmt.Sprintf("%ds", sleepSeconds),
	)
	assert.NoError(t, err)
	time.Sleep(sleepDuration)

	up, err = mon.Status(testServiceName)
	assert.NoError(t, err)
	assert.False(t, up)
}

// TestMinMonitorNonExistantStatus verifies that a MinMonitor generated by
// NewMinMonitor will give the expected status for a service that is up.
func TestMinMonitorNonExistantStatus(t *testing.T) {
	testServiceName := "test"

	mon := MinMonitor{}

	up, err := mon.Status(testServiceName)
	assert.Error(t, err)
	assert.Equal(t, UnknownServiceError, err)
	assert.False(t, up)
}

// TestMinMonitorNonExistantSetStatusUp verifies that a MinMonitor generated by
// NewMinMonitor will give the expected status for a service that is up.
func TestMinMonitorNonExistantSetStatusUp(t *testing.T) {
	testServiceName := "test"

	mon := MinMonitor{}

	err := mon.SetStatusUp(testServiceName)
	assert.Error(t, err)
	assert.Equal(t, UnknownServiceError, err)
}

// TestMinMonitorNonExistantReprobe verifies that a MinMonitor generated by
// NewMinMonitor will give the expected status for a service that is up.
func TestMinMonitorNonExistantReprobe(t *testing.T) {
	testServiceName := "test"

	mon := MinMonitor{}

	up, err := mon.Reprobe(testServiceName)
	assert.Error(t, err)
	assert.Equal(t, UnknownServiceError, err)
	assert.False(t, up)
}

// TestMinMonitorAddExistant verifies that a MinMonitor generated by
// NewMinMonitor will give the expected status for a service that is up.
func TestMinMonitorAddExistant(t *testing.T) {
	testServiceName := "test"
	testHost := "localhost"
	testProtocol := "tcp"
	gracePeriod := time.Duration(0)

	landingPipeline := falcore.NewPipeline()
	landingPipeline.Upstream.PushBack(&util.LandingFilter{})
	landingServer := falcore.NewServer(0, landingPipeline)
	go serveLandingPage(landingServer)
	defer landingServer.StopAccepting()

	<- landingServer.AcceptReady

	svc, err := NewMinMonitorredService(
		testHost,
		landingServer.Port(),
		testProtocol,
		gracePeriod,
		nil,
		nil,
		nil,
	)
	assert.NoError(t, err)
	mon := MinMonitor{}
	err = mon.Add(
		testServiceName,
		svc,
	)
	assert.NoError(t, err)

	svc2, err := NewMinMonitorredService(
		testHost,
		landingServer.Port() + 1,
		testProtocol,
		gracePeriod,
		nil,
		nil,
		nil,
	)
	assert.NoError(t, err)
	err = mon.Add(
		testServiceName,
		svc2,
	)
	assert.Error(t, err)
	assert.Equal(t, DuplicateServiceRegistrationError, err)
}

func TestMonitorFilterUp(t *testing.T) {
	request, err := http.NewRequest("GET", "http://localhost", nil)
	assert.NoError(t, err)

	testServiceName := "test"
	testHost := "localhost"
	testProtocol := "tcp"
	gracePeriod := time.Duration(0)

	landingPipeline := falcore.NewPipeline()
	landingPipeline.Upstream.PushBack(&util.LandingFilter{})
	landingServer := falcore.NewServer(0, landingPipeline)
	go serveLandingPage(landingServer)
	defer landingServer.StopAccepting()

	<- landingServer.AcceptReady

	service, err := NewMinMonitorredService(
		testHost,
		landingServer.Port(),
		testProtocol,
		gracePeriod,
		nil,
		nil,
		nil,
	)
	assert.NoError(t, err)
	mon := MinMonitor{}
	err = mon.Add(
		testServiceName,
		service,
	)
	assert.NoError(t, err)

	filter, err := mon.NewMinMonitorFilter(testServiceName)
	assert.NoError(t, err)

	_, response := falcore.TestWithRequest(
		request,
		filter,
		nil,
	)

	assert.Equal(t, 200, response.StatusCode)
	contents, err := ioutil.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.True(
		t,
		strings.Contains(string(contents), "Pullcord Landing Page"),
		"content is: " + string(contents),
	)
}

func TestMonitorFilterDown(t *testing.T) {
	request, err := http.NewRequest("GET", "http://localhost", nil)
	assert.NoError(t, err)

	testServiceName := "test"
	testHost := "localhost"
	testProtocol := "tcp"
	gracePeriod := time.Duration(0)

	server, err := net.Listen(testProtocol, ":0")
	assert.NoError(t, err)
	_, rawPort, err := net.SplitHostPort(server.Addr().String())
	assert.NoError(t, err)
	testPort, err := strconv.Atoi(rawPort)
	assert.NoError(t, err)
	err = server.Close()
	assert.NoError(t, err)

	svc, err := NewMinMonitorredService(
		testHost,
		testPort,
		testProtocol,
		gracePeriod,
		nil,
		nil,
		nil,
	)
	assert.NoError(t, err)
	mon := MinMonitor{}
	err = mon.Add(
		testServiceName,
		svc,
	)
	assert.NoError(t, err)

	filter, err := mon.NewMinMonitorFilter(testServiceName)
	assert.NoError(t, err)

	_, response := falcore.TestWithRequest(
		request,
		filter,
		nil,
	)

	assert.Equal(t, 503, response.StatusCode)
	contents, err := ioutil.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.True(
		t,
		strings.Contains(string(contents), "Service Not Ready"),
		"content is: " + string(contents),
	)
}

func TestMonitorFilterBadServiceName(t *testing.T) {
	mon := MinMonitor{}
	_, err := mon.NewMinMonitorFilter("unknown_service")
	assert.Error(t, err)
}

type counterTriggerHandler struct {
	count int
}

func (th *counterTriggerHandler) Trigger() error {
	if th.count < 0 {
		return errors.New("this trigger always errors")
	} else {
		th.count += 1
		return nil
	}
}

func TestMonitorFilterUpTriggers(t *testing.T) {
	request, err := http.NewRequest("GET", "http://localhost", nil)
	assert.NoError(t, err)

	testServiceName := "test"
	testHost := "localhost"
	testProtocol := "tcp"
	gracePeriod := time.Duration(0)

	landingPipeline := falcore.NewPipeline()
	landingPipeline.Upstream.PushBack(&util.LandingFilter{})
	landingServer := falcore.NewServer(0, landingPipeline)
	go serveLandingPage(landingServer)
	defer landingServer.StopAccepting()

	<- landingServer.AcceptReady

	onDown := &counterTriggerHandler{}
	onUp := &counterTriggerHandler{}
	always := &counterTriggerHandler{}

	service, err := NewMinMonitorredService(
		testHost,
		landingServer.Port(),
		testProtocol,
		gracePeriod,
		onDown,
		onUp,
		always,
	)
	assert.NoError(t, err)
	mon := MinMonitor{}
	err = mon.Add(
		testServiceName,
		service,
	)
	assert.NoError(t, err)

	filter, err := mon.NewMinMonitorFilter(
		testServiceName,
	)
	assert.NoError(t, err)

	_, response := falcore.TestWithRequest(
		request,
		filter,
		nil,
	)

	assert.Equal(t, 200, response.StatusCode)
	contents, err := ioutil.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.True(
		t,
		strings.Contains(string(contents), "Pullcord Landing Page"),
		"content is: " + string(contents),
	)
	assert.Equal(t, 0, onDown.count)
	assert.Equal(t, 1, onUp.count)
	assert.Equal(t, 1, always.count)
}

func TestMonitorFilterDownTriggers(t *testing.T) {
	request, err := http.NewRequest("GET", "http://localhost", nil)
	assert.NoError(t, err)

	testServiceName := "test"
	testHost := "localhost"
	testProtocol := "tcp"
	gracePeriod := time.Duration(0)

	server, err := net.Listen(testProtocol, ":0")
	assert.NoError(t, err)
	_, rawPort, err := net.SplitHostPort(server.Addr().String())
	assert.NoError(t, err)
	testPort, err := strconv.Atoi(rawPort)
	assert.NoError(t, err)
	err = server.Close()
	assert.NoError(t, err)

	onDown := &counterTriggerHandler{}
	onUp := &counterTriggerHandler{}
	always := &counterTriggerHandler{}

	svc, err := NewMinMonitorredService(
		testHost,
		testPort,
		testProtocol,
		gracePeriod,
		onDown,
		onUp,
		always,
	)
	assert.NoError(t, err)
	mon := MinMonitor{}
	err = mon.Add(
		testServiceName,
		svc,
	)
	assert.NoError(t, err)

	filter, err := mon.NewMinMonitorFilter(
		testServiceName,
	)
	assert.NoError(t, err)

	_, response := falcore.TestWithRequest(
		request,
		filter,
		nil,
	)

	assert.Equal(t, 503, response.StatusCode)
	contents, err := ioutil.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.True(
		t,
		strings.Contains(string(contents), "Service Not Ready"),
		"content is: " + string(contents),
	)
	assert.Equal(t, 1, onDown.count)
	assert.Equal(t, 0, onUp.count)
	assert.Equal(t, 1, always.count)
}

func TestMonitorFilterUpOnUpTriggerError(t *testing.T) {
	request, err := http.NewRequest("GET", "http://localhost", nil)
	assert.NoError(t, err)

	testServiceName := "test"
	testHost := "localhost"
	testProtocol := "tcp"
	gracePeriod := time.Duration(0)

	landingPipeline := falcore.NewPipeline()
	landingPipeline.Upstream.PushBack(&util.LandingFilter{})
	landingServer := falcore.NewServer(0, landingPipeline)
	go serveLandingPage(landingServer)
	defer landingServer.StopAccepting()

	<- landingServer.AcceptReady

	onDown := &counterTriggerHandler{}
	onUp := &counterTriggerHandler{-1}
	always := &counterTriggerHandler{}

	service, err := NewMinMonitorredService(
		testHost,
		landingServer.Port(),
		testProtocol,
		gracePeriod,
		onDown,
		onUp,
		always,
	)
	assert.NoError(t, err)
	mon := MinMonitor{}
	err = mon.Add(
		testServiceName,
		service,
	)
	assert.NoError(t, err)

	filter, err := mon.NewMinMonitorFilter(
		testServiceName,
	)
	assert.NoError(t, err)

	_, response := falcore.TestWithRequest(
		request,
		filter,
		nil,
	)

	assert.Equal(t, 500, response.StatusCode)
	contents, err := ioutil.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.True(
		t,
		strings.Contains(string(contents), "Internal Server Error"),
		"content is: " + string(contents),
	)
	assert.Equal(t, -1, onUp.count)
}

func TestMonitorFilterDownOnDownTriggerError(t *testing.T) {
	request, err := http.NewRequest("GET", "http://localhost", nil)
	assert.NoError(t, err)

	testServiceName := "test"
	testHost := "localhost"
	testProtocol := "tcp"
	gracePeriod := time.Duration(0)

	server, err := net.Listen(testProtocol, ":0")
	assert.NoError(t, err)
	_, rawPort, err := net.SplitHostPort(server.Addr().String())
	assert.NoError(t, err)
	testPort, err := strconv.Atoi(rawPort)
	assert.NoError(t, err)
	err = server.Close()
	assert.NoError(t, err)

	onDown := &counterTriggerHandler{-1}
	onUp := &counterTriggerHandler{}
	always := &counterTriggerHandler{}

	svc, err := NewMinMonitorredService(
		testHost,
		testPort,
		testProtocol,
		gracePeriod,
		onDown,
		onUp,
		always,
	)
	assert.NoError(t, err)
	mon := MinMonitor{}
	err = mon.Add(
		testServiceName,
		svc,
	)
	assert.NoError(t, err)

	filter, err := mon.NewMinMonitorFilter(
		testServiceName,
	)
	assert.NoError(t, err)

	_, response := falcore.TestWithRequest(
		request,
		filter,
		nil,
	)

	assert.Equal(t, 500, response.StatusCode)
	contents, err := ioutil.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.True(
		t,
		strings.Contains(string(contents), "Internal Server Error"),
		"content is: " + string(contents),
	)
	assert.Equal(t, -1, onDown.count)
}

func TestMonitorFilterDownAlwaysTriggerError(t *testing.T) {
	request, err := http.NewRequest("GET", "http://localhost", nil)
	assert.NoError(t, err)

	testServiceName := "test"
	testHost := "localhost"
	testProtocol := "tcp"
	gracePeriod := time.Duration(0)

	server, err := net.Listen(testProtocol, ":0")
	assert.NoError(t, err)
	_, rawPort, err := net.SplitHostPort(server.Addr().String())
	assert.NoError(t, err)
	testPort, err := strconv.Atoi(rawPort)
	assert.NoError(t, err)
	err = server.Close()
	assert.NoError(t, err)

	onDown := &counterTriggerHandler{}
	onUp := &counterTriggerHandler{}
	always := &counterTriggerHandler{-1}

	svc, err := NewMinMonitorredService(
		testHost,
		testPort,
		testProtocol,
		gracePeriod,
		onDown,
		onUp,
		always,
	)
	assert.NoError(t, err)
	mon := MinMonitor{}
	err = mon.Add(
		testServiceName,
		svc,
	)
	assert.NoError(t, err)

	filter, err := mon.NewMinMonitorFilter(
		testServiceName,
	)
	assert.NoError(t, err)

	_, response := falcore.TestWithRequest(
		request,
		filter,
		nil,
	)

	assert.Equal(t, 500, response.StatusCode)
	contents, err := ioutil.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.True(
		t,
		strings.Contains(string(contents), "Internal Server Error"),
		"content is: " + string(contents),
	)
	assert.Equal(t, -1, always.count)
}

func TestMinMonitorFromConfig(t *testing.T) {
	test := configutil.ConfigTest{
		ResourceType: "minmonitorredservice",
		SyntacticallyBad: []configutil.ConfigTestData{
			configutil.ConfigTestData{
				Data: "",
				Explanation: "empty config",
			},
			configutil.ConfigTestData{
				Data: "{}",
				Explanation: "empty object",
			},
			configutil.ConfigTestData{
				Data: "null",
				Explanation: "null config",
			},
			configutil.ConfigTestData{
				Data: "42",
				Explanation: "numeric config",
			},
		},
		Good: []configutil.ConfigTestData{
			configutil.ConfigTestData{
				Data: `{
					"address": "127.0.0.1",
					"port": 80,
					"protocol": "http",
					"graceperiod": "1s"
				}`,
				Explanation: "basic valid monitor config",
			},
		},
	}
	test.Run(t)
}

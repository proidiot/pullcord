package authentication

import (
	"fmt"
	"github.com/fitstar/falcore"
	"github.com/stretchr/testify/assert"
	// "github.com/stuphlabs/pullcord"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestInitialLoginPage(t *testing.T) {
	/* setup */
	testUser := "testUser"
	testPassword := "P@ssword1"

	downstreamFilter := falcore.NewRequestFilter(
		func (request *falcore.Request) *http.Response {
			return falcore.StringResponse(
				request.HttpRequest,
				200,
				nil,
				"<html><body><p>logged in</p></body></html>",
			)
		},
	)
	sessionHandler := NewMinSessionHandler(
		"testSessionHandler",
		"/",
		"example.com",
	)
	passwordChecker := NewInMemPwdStore()
	err := passwordChecker.SetPassword(
		testUser,
		testPassword,
		Pbkdf2MinIterations,
	)
	assert.NoError(t, err)

	request, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)

	/* run */
	var handler LoginHandler
	handler.Identifier = "testLoginHandler"
	handler.PasswordChecker = passwordChecker
	handler.Downstream = downstreamFilter
	loginHandler := NewLoginFilter(
		sessionHandler,
		handler,
	)
	_, response := falcore.TestWithRequest(request, loginHandler, nil)

	/* check */
	assert.Equal(t, 200, response.StatusCode)

	content, err := ioutil.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.True(
		t,
		strings.Contains(string(content), "xsrf-testLoginHandler"),
		"content is: " + string(content),
	)

	assert.NotEmpty(t, response.Header["Set-Cookie"])
}

func TestNoXsrfLoginPage(t *testing.T) {
	/* setup */
	testUser := "testUser"
	testPassword := "P@ssword1"

	downstreamFilter := falcore.NewRequestFilter(
		func (request *falcore.Request) *http.Response {
			return falcore.StringResponse(
				request.HttpRequest,
				200,
				nil,
				"<html><body><p>logged in</p></body></html>",
			)
		},
	)
	sessionHandler := NewMinSessionHandler(
		"testSessionHandler",
		"/",
		"example.com",
	)
	passwordChecker := NewInMemPwdStore()
	err := passwordChecker.SetPassword(
		testUser,
		testPassword,
		Pbkdf2MinIterations,
	)
	assert.NoError(t, err)

	request1, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)

	postdata2 := url.Values{}
	request2, err := http.NewRequest(
		"POST",
		"/",
		strings.NewReader(postdata2.Encode()),
	)
	assert.NoError(t, err)

	/* run */
	var handler LoginHandler
	handler.Identifier = "testLoginHandler"
	handler.PasswordChecker = passwordChecker
	handler.Downstream = downstreamFilter
	loginHandler := NewLoginFilter(
		sessionHandler,
		handler,
	)

	_, response1 := falcore.TestWithRequest(request1, loginHandler, nil)
	assert.Equal(t, 200, response1.StatusCode)
	assert.NotEmpty(t, response1.Header["Set-Cookie"])
	for _, cke := range response1.Cookies() {
		request2.AddCookie(cke)
	}

	_, response2 := falcore.TestWithRequest(request2, loginHandler, nil)

	/* check */
	assert.Equal(t, 200, response2.StatusCode)

	content, err := ioutil.ReadAll(response2.Body)
	assert.NoError(t, err)
	assert.True(
		t,
		strings.Contains(string(content), "Invalid credentials"),
		"content is: " + string(content),
	)
}

func TestBadXsrfLoginPage(t *testing.T) {
	/* setup */
	testUser := "testUser"
	testPassword := "P@ssword1"

	downstreamFilter := falcore.NewRequestFilter(
		func (request *falcore.Request) *http.Response {
			return falcore.StringResponse(
				request.HttpRequest,
				200,
				nil,
				"<html><body><p>logged in</p></body></html>",
			)
		},
	)
	sessionHandler := NewMinSessionHandler(
		"testSessionHandler",
		"/",
		"example.com",
	)
	passwordChecker := NewInMemPwdStore()
	err := passwordChecker.SetPassword(
		testUser,
		testPassword,
		Pbkdf2MinIterations,
	)
	assert.NoError(t, err)

	request1, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)

	postdata2 := url.Values{}
	postdata2.Add("xsrf-testLoginHandler", "")
	assert.Empty(t, postdata2.Encode())
	request2, err := http.NewRequest(
		"POST",
		"/",
		strings.NewReader(postdata2.Encode()),
	)
	assert.NoError(t, err)

	/* run */
	var handler LoginHandler
	handler.Identifier = "testLoginHandler"
	handler.PasswordChecker = passwordChecker
	handler.Downstream = downstreamFilter
	loginHandler := NewLoginFilter(
		sessionHandler,
		handler,
	)

	_, response1 := falcore.TestWithRequest(request1, loginHandler, nil)
	assert.Equal(t, 200, response1.StatusCode)
	assert.NotEmpty(t, response1.Header["Set-Cookie"])
	for _, cke := range response1.Cookies() {
		request2.AddCookie(cke)
	}
	log().Debug(fmt.Sprintf("request: %v", request2))

	_, response2 := falcore.TestWithRequest(request2, loginHandler, nil)

	/* check */
	assert.Equal(t, 200, response2.StatusCode)

	content, err := ioutil.ReadAll(response2.Body)
	assert.NoError(t, err)
	assert.True(
		t,
		strings.Contains(string(content), "I nvalid credentials"),
		"content is: " + string(content),
	)
}


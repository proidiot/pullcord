package pullcord

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/dustin/randbo"
	"github.com/fitstar/falcore"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const cookieMaskTestCookieNameRandSize = 16
const cookieMaskTestCookieValueRandSize = 64

var exampleCookieValueRegex = regexp.MustCompile(
	"^[0-9A-Fa-f]{" +
	strconv.Itoa(cookieMaskTestCookieValueRandSize*2) +
	"}$",
)
var randgen = randbo.New()

// gostring is a testing helper function that serializes any object.
func gostring(i interface{}) string {
	return fmt.Sprintf("%#v", i)
}

var cookieMaskTestPage = falcore.NewRequestFilter(
	func(req *falcore.Request) *http.Response {
		var content = "<html><body><h1>cookies</h1><ul id=\"cke\">"
		for _, cke := range req.HttpRequest.Cookies() {
			content += "<li>" + cke.String() + "</li>"
		}
		content += "</ul><h1>context</h1><ul id="\"ctx\">"
		for name, ctx := range req.Context {
			content +=
				"<li>" +
				name +
				": " +
				gostring(ctx) +
				"</li>"
		}
		content += "</ul></body></html>"
		return falcore.StringResponse(
			req.HttpRequest,
			200,
			nil,
			content,
		)
	},
)

var errorPage = falcore.NewRequestFilter(
	func(req *falcore.Request) *http.Response {
		// BUG(proidiot) issue-#45: Does not print context or cookies
		return falcore.StringResponse(
			req.HttpRequest,
			500,
			nil,
			"<html><body><p>" +
			"internal server error" +
			"</p></body></html>",
		)
	},
)

// exampleCookieGen is a testing helper function that generates a randomized
// cookie name and value pair but with the given variant string being in a
// predictable location in the cookie name so it will be easy to identify with
// regular expressions.
func exampleCookieGen(variant string) (result http.Cookie) {
	nbytes := make([]byte, cookieMaskTestCookieNameRandSize)
	vbytes := make([]byte, cookieMaskTestCookieValueRandSize)
	randgen.Read(nbytes)
	randgen.Read(vbytes)
	result.Name = "example-" + variant + "-" + hex.EncodeToString(nbytes)
	result.Value = hex.EncodeToString(vbytes)
	return result
}

// exampleCookieMaskGen is a testing helper function that will generate a cookie
// mask function that will mask cookies generated by exampleCookieGen given the
// save variant string.
//
// This function also adds items into the context to simulate the behavior of an
// associated session handler, sets new cookies from exampleCookieGen if no
// matching cookie was found, and even generates an error if the variant string
// is "error". As such, it should give a more complete, if very crude,
// approximation of a real-world implementation of a cookie mask.
func exampleCookieMaskGen(variant string) func(inCookies []*http.Cookie) (
	fwdCookies []*http.Cookie,
	setCookies []*http.Cookie,
	context map[string]interface{},
	err error,
) {
	var exampleCookieNameRegex = regexp.MustCompile(
		"^example-" +
		variant +
		"-[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieNameRandSize*2) +
		"}$",
	)

	return func(inCookies []*http.Cookie) (
		fwdCookies []*http.Cookie,
		setCookies []*http.Cookie,
		ctx map[string]interface{},
		err error,
	) {
		if variant == "error" {
			err = errors.New("got an error")
		}
		ctx = make(map[string]interface{})
		found := false
		for _, cke := range inCookies {
			if exampleCookieNameRegex.MatchString(cke.Name) &&
			    exampleCookieValueRegex.MatchString(cke.Value) {
				found = true
				ctx["example-"+variant+"-cookie-found"] = "true"
			} else {
				fwdCookies = append(fwdCookies, cke)
			}
		}
		if !found {
			setCookies = make([]*http.Cookie, 1)
			sub_cke := exampleCookieGen(variant)
			setCookies[0] = &sub_cke
			ctx["example-"+variant+"-cookie-found"] = "false"
		}

		return fwdCookies, setCookies, ctx, err
	}
}

// TestCookiemaskCookieless verifies that a Falcore RequestFilter generated by
// NewCookiemaskFilter behaves as expected when given a request that contains no
// cookies. Specifically, this function verifies that no error occurred, that
// the next RequestFilter in the chain did not receive an indication that a
// maskable cookie was found, and that a new maskable cookie of the expected
// variant is being set.
func TestCookiemaskCookieless(t *testing.T) {
	/* setup */
	cookieRegex := regexp.MustCompile(
		"^example-test-[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieNameRandSize*2) +
		"}=[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieValueRandSize*2) +
		"}",
	)
	request, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)

	/* run */
	_, response := falcore.TestWithRequest(
		request,
		NewCookiemaskFilter(
			exampleCookieMaskGen("test"),
			cookieMaskTestPage,
			errorPage,
		),
		nil,
	)

	/* check */
	assert.Equal(t, 200, response.StatusCode)

	contents, err := ioutil.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.True(
		t,
		strings.Contains(
			string(contents),
			"example-test-cookie-found: \"false\"",
		),
		"contents are " + string(contents),
	)

	new_cookie_set := false
	for _, cke_str := range response.Header["Set-Cookie"] {
		if cookieRegex.MatchString(cke_str) {
			new_cookie_set = true
		}
	}
	assert.True(
		t,
		new_cookie_set,
		"regex didn't match any of " +
		gostring(response.Header["Set-Cookie"]),
	)
}

// TestCookiemaskNoMasking verifies that a Falcore RequestFilter generated by 
// NewCookiemaskFilter behaves as expected when given a request that contains
// only a cookie that is not supposed to be masked. Specifically, this function
// verifies that no error occurred, that the submitted cookie which was not
// supposed to be masked did in fact make it safely to the next RequestFilter in
// the chain, that the next RequestFilter in the chain did not receive an
// indication that a maskable cookie was found, and that a new maskable cookie
// of the expected variant is being set.
func TestCookiemaskNoMasking(t *testing.T) {
	/* setup */
	cookieRegex := regexp.MustCompile(
		"^example-test-[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieNameRandSize*2) +
		"}=[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieValueRandSize*2) +
		"}",
	)
	request, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)
	cookie := exampleCookieGen("foo")
	request.AddCookie(&cookie)

	/* run */
	_, response := falcore.TestWithRequest(
		request,
		NewCookiemaskFilter(
			exampleCookieMaskGen("test"),
			cookieMaskTestPage,
			errorPage,
		),
		nil,
	)

	/* check */
	assert.Equal(t, 200, response.StatusCode)

	contents, err := ioutil.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.True(t, strings.Contains(string(contents), cookie.String()))
	assert.True(
		t,
		strings.Contains(
			string(contents),
			"example-test-cookie-found: \"false\"",
		),
		"contents are " + string(contents),
	)

	new_cookie_set := false
	for _, cke_str := range response.Header["Set-Cookie"] {
		if cookieRegex.MatchString(cke_str) {
			new_cookie_set = true
		}
	}
	assert.True(
		t,
		new_cookie_set,
		"regex didn't match any of " +
		gostring(response.Header["Set-Cookie"]),
	)
}

// TestCookiemaskError verifies that a Falcore RequestFilter generated by 
// NewCookiemaskFilter behaves as expected when the cookie mask function throws
// an error. Specifically, this function verifies that an internal server error
// is sent in the response, and that no cookie is being set.
func TestCookiemaskError(t *testing.T) {
	/* setup */
	cookieRegex := regexp.MustCompile(
		"^example-test-[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieNameRandSize*2) +
		"}=[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieValueRandSize*2) +
		"}",
	)
	request, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)
	cookie := exampleCookieGen("foo")
	request.AddCookie(&cookie)

	/* run */
	_, response := falcore.TestWithRequest(
		request,
		NewCookiemaskFilter(
			exampleCookieMaskGen("error"),
			cookieMaskTestPage,
			errorPage,
		),
		nil,
	)

	/* check */
	assert.Equal(t, 500, response.StatusCode)

	new_cookie_set := false
	for _, cke_str := range response.Header["Set-Cookie"] {
		if cookieRegex.MatchString(cke_str) {
			new_cookie_set = true
		}
	}
	assert.False(
		t,
		new_cookie_set,
		"regex matched one of " +
		gostring(response.Header["Set-Cookie"]),
	)
}

// TestCookiemaskMasking verifies that a Falcore RequestFilter generated by 
// NewCookiemaskFilter behaves as expected when given a request that contains
// a cookie that is supposed to be masked. Specifically, this function verifies
// that no error occurred, that the submitted cookie was not forwarded to the
// next RequestFilter in the chain, that the next RequestFilter in the chain did
// indeed receive an indication that a maskable cookie was found, and that no
// new maskable cookie of the expected variant is being set.
func TestCookiemaskMasking(t *testing.T) {
	/* setup */
	cookieRegex := regexp.MustCompile(
		"^example-test-[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieNameRandSize*2) +
		"}=[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieValueRandSize*2) +
		"}",
	)
	request, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)
	cookie := exampleCookieGen("test")
	request.AddCookie(&cookie)

	/* run */
	_, response := falcore.TestWithRequest(
		request,
		NewCookiemaskFilter(
			exampleCookieMaskGen("test"),
			cookieMaskTestPage,
			errorPage,
		),
		nil,
	)

	/* check */
	assert.Equal(t, 200, response.StatusCode)

	contents, err := ioutil.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.False(t, strings.Contains(string(contents), cookie.String()))
	assert.True(
		t,
		strings.Contains(
			string(contents),
			"example-test-cookie-found: \"true\"",
		),
		"contents are " + string(contents),
	)

	new_cookie_set := false
	for _, cke_str := range response.Header["Set-Cookie"] {
		if cookieRegex.MatchString(cke_str) {
			new_cookie_set = true
		}
	}
	assert.False(
		t,
		new_cookie_set,
		"regex matched one of " +
		gostring(response.Header["Set-Cookie"]),
	)
}

// TestDoubleCookiemaskNoMasking verifies that a chain of two Falcore
// RequestFilters each generated by NewCookiemaskFilter behaves as expected when
// given a request that contains two cookies that are not supposed to be masked.
// Specifically, this function verifies that no error occurred, that both the
// submitted cookies which were not supposed to be masked did in fact make it
// safely to the last RequestFilter in the chain, that the last RequestFilter in
// the chain did not receive an indication that any maskable cookies were found,
// and that a new maskable cookie of each of the two expected variants are being
// set.
func TestDoubleCookiemaskNoMasking(t *testing.T) {
	/* setup */
	cookieRegex1 := regexp.MustCompile(
		"^example-test1-[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieNameRandSize*2) +
		"}=[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieValueRandSize*2) +
		"}",
	)
	cookieRegex2 := regexp.MustCompile(
		"^example-test2-[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieNameRandSize*2) +
		"}=[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieValueRandSize*2) +
		"}",
	)
	request, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)
	cookie1 := exampleCookieGen("foo1")
	cookie2 := exampleCookieGen("foo2")
	request.AddCookie(&cookie1)
	request.AddCookie(&cookie2)

	/* run */
	_, response := falcore.TestWithRequest(
		request,
		NewCookiemaskFilter(
			exampleCookieMaskGen("test1"),
			NewCookiemaskFilter(
				exampleCookieMaskGen("test2"),
				cookieMaskTestPage,
				errorPage,
			),
			errorPage,
		),
		nil,
	)

	/* check */
	assert.Equal(t, 200, response.StatusCode)

	contents, err := ioutil.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.True(t, strings.Contains(string(contents), cookie1.String()))
	assert.True(t, strings.Contains(string(contents), cookie2.String()))
	assert.True(
		t,
		strings.Contains(
			string(contents),
			"example-test1-cookie-found: \"false\"",
		),
		"contents are " + string(contents),
	)
	assert.True(
		t,
		strings.Contains(
			string(contents),
			"example-test2-cookie-found: \"false\"",
		),
		"contents are " + string(contents),
	)

	new_cookie1_set := false
	new_cookie2_set := false
	for _, cke_str := range response.Header["Set-Cookie"] {
		if cookieRegex1.MatchString(cke_str) {
			new_cookie1_set = true
		} else if cookieRegex2.MatchString(cke_str) {
			new_cookie2_set = true
		}
	}
	assert.True(
		t,
		new_cookie1_set,
		"regex1 didn't match any of " +
		gostring(response.Header["Set-Cookie"]),
	)
	assert.True(
		t,
		new_cookie2_set,
		"regex2 didn't match any of " +
		gostring(response.Header["Set-Cookie"]),
	)
}

// TestDoubleCookiemaskTopMasking verifies that a chain of two Falcore
// RequestFilters each generated by NewCookiemaskFilter behaves as expected when
// given a request that contains one cookie that is to be masked by the first
// RequestFilter in the chain and a second cookie that is not to be masked by
// either RequestFilter. Specifically, this function verifies that no error
// occurred, that only the submitted cookie which was not supposed to be masked
// did in fact make it safely to the last RequestFilter in the chain, that the
// last RequestFilter in the chain received an indication that the particular
// maskable cookie was found, and that a new maskable cookie of the expected
// variant for the second RequestFilter in the chain is being set.
func TestDoubleCookiemaskTopMasking(t *testing.T) {
	/* setup */
	cookieRegex1 := regexp.MustCompile(
		"^example-test1-[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieNameRandSize*2) +
		"}=[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieValueRandSize*2) +
		"}",
	)
	cookieRegex2 := regexp.MustCompile(
		"^example-test2-[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieNameRandSize*2) +
		"}=[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieValueRandSize*2) +
		"}",
	)
	request, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)
	cookie1 := exampleCookieGen("test1")
	cookie2 := exampleCookieGen("foo")
	request.AddCookie(&cookie1)
	request.AddCookie(&cookie2)

	/* run */
	_, response := falcore.TestWithRequest(
		request,
		NewCookiemaskFilter(
			exampleCookieMaskGen("test1"),
			NewCookiemaskFilter(
				exampleCookieMaskGen("test2"),
				cookieMaskTestPage,
				errorPage,
			),
			errorPage,
		),
		nil,
	)

	/* check */
	assert.Equal(t, 200, response.StatusCode)

	contents, err := ioutil.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.False(t, strings.Contains(string(contents), cookie1.String()))
	assert.True(t, strings.Contains(string(contents), cookie2.String()))
	assert.True(
		t,
		strings.Contains(
			string(contents),
			"example-test1-cookie-found: \"true\"",
		),
		"contents are " + string(contents),
	)
	assert.True(
		t,
		strings.Contains(
			string(contents),
			"example-test2-cookie-found: \"false\"",
		),
		"contents are " + string(contents),
	)

	new_cookie1_set := false
	new_cookie2_set := false
	for _, cke_str := range response.Header["Set-Cookie"] {
		if cookieRegex1.MatchString(cke_str) {
			new_cookie1_set = true
		} else if cookieRegex2.MatchString(cke_str) {
			new_cookie2_set = true
		}
	}
	assert.False(
		t,
		new_cookie1_set,
		"regex1 matched one of " +
		gostring(response.Header["Set-Cookie"]),
	)
	assert.True(
		t,
		new_cookie2_set,
		"regex2 didn't match any of " +
		gostring(response.Header["Set-Cookie"]),
	)
}

// TestDoubleCookiemaskBottomMasking verifies that a chain of two Falcore
// RequestFilters each generated by NewCookiemaskFilter behaves as expected when
// given a request that contains one cookie that is to be masked by the second
// RequestFilter in the chain and a second cookie that is not to be masked by
// either RequestFilter. Specifically, this function verifies that no error
// occurred, that only the submitted cookie which was not supposed to be masked
// did in fact make it safely to the last RequestFilter in the chain, that the
// last RequestFilter in the chain received an indication that the particular
// maskable cookie was found, and that a new maskable cookie of the expected
// variant for the first RequestFilter in the chain is being set.
func TestDoubleCookiemaskBottomMasking(t *testing.T) {
	/* setup */
	cookieRegex1 := regexp.MustCompile(
		"^example-test1-[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieNameRandSize*2) +
		"}=[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieValueRandSize*2) +
		"}",
	)
	cookieRegex2 := regexp.MustCompile(
		"^example-test2-[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieNameRandSize*2) +
		"}=[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieValueRandSize*2) +
		"}",
	)
	request, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)
	cookie1 := exampleCookieGen("foo")
	cookie2 := exampleCookieGen("test2")
	request.AddCookie(&cookie1)
	request.AddCookie(&cookie2)

	/* run */
	_, response := falcore.TestWithRequest(
		request,
		NewCookiemaskFilter(
			exampleCookieMaskGen("test1"),
			NewCookiemaskFilter(
				exampleCookieMaskGen("test2"),
				cookieMaskTestPage,
				errorPage,
			),
			errorPage,
		),
		nil,
	)

	/* check */
	assert.Equal(t, 200, response.StatusCode)

	contents, err := ioutil.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.True(t, strings.Contains(string(contents), cookie1.String()))
	assert.False(t, strings.Contains(string(contents), cookie2.String()))
	assert.True(
		t,
		strings.Contains(
			string(contents),
			"example-test1-cookie-found: \"false\"",
		),
		"contents are " + string(contents),
	)
	assert.True(
		t,
		strings.Contains(
			string(contents),
			"example-test2-cookie-found: \"true\"",
		),
		"contents are " + string(contents),
	)

	new_cookie1_set := false
	new_cookie2_set := false
	for _, cke_str := range response.Header["Set-Cookie"] {
		if cookieRegex1.MatchString(cke_str) {
			new_cookie1_set = true
		} else if cookieRegex2.MatchString(cke_str) {
			new_cookie2_set = true
		}
	}
	assert.True(
		t,
		new_cookie1_set,
		"regex1 didn't match any of " +
		gostring(response.Header["Set-Cookie"]),
	)
	assert.False(
		t,
		new_cookie2_set,
		"regex2 matched one of " +
		gostring(response.Header["Set-Cookie"]),
	)
}

// TestDoubleCookiemaskBothMasking verifies that a chain of two Falcore
// RequestFilters each generated by NewCookiemaskFilter behaves as expected when
// given a request that contains two cookies that are each supposed to be
// masked. Specifically, this function verifies that no error occurred, that
// neither of the submitted cookies which were supposed to be masked did in fact
// make it safely to the last RequestFilter in the chain, that the last
// RequestFilter the chain received indications that both of the maskable
// cookies were found, and that no new maskable cookie is being set.
func TestDoubleCookiemaskBothMasking(t *testing.T) {
	/* setup */
	cookieRegex1 := regexp.MustCompile(
		"^example-test1-[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieNameRandSize*2) +
		"}=[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieValueRandSize*2) +
		"}",
	)
	cookieRegex2 := regexp.MustCompile(
		"^example-test2-[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieNameRandSize*2) +
		"}=[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieValueRandSize*2) +
		"}",
	)
	request, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)
	cookie1 := exampleCookieGen("test1")
	cookie2 := exampleCookieGen("test2")
	request.AddCookie(&cookie1)
	request.AddCookie(&cookie2)

	/* run */
	_, response := falcore.TestWithRequest(
		request,
		NewCookiemaskFilter(
			exampleCookieMaskGen("test1"),
			NewCookiemaskFilter(
				exampleCookieMaskGen("test2"),
				cookieMaskTestPage,
				errorPage,
			),
			errorPage,
		),
		nil,
	)

	/* check */
	assert.Equal(t, 200, response.StatusCode)

	contents, err := ioutil.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.False(t, strings.Contains(string(contents), cookie1.String()))
	assert.False(t, strings.Contains(string(contents), cookie2.String()))
	assert.True(
		t,
		strings.Contains(
			string(contents),
			"example-test1-cookie-found: \"true\"",
		),
		"contents are " + string(contents),
	)
	assert.True(
		t,
		strings.Contains(
			string(contents),
			"example-test2-cookie-found: \"true\"",
		),
		"contents are " + string(contents),
	)

	new_cookie1_set := false
	new_cookie2_set := false
	for _, cke_str := range response.Header["Set-Cookie"] {
		if cookieRegex1.MatchString(cke_str) {
			new_cookie1_set = true
		} else if cookieRegex2.MatchString(cke_str) {
			new_cookie2_set = true
		}
	}
	assert.False(
		t,
		new_cookie1_set,
		"regex1 matched one of " +
		gostring(response.Header["Set-Cookie"]),
	)
	assert.False(
		t,
		new_cookie2_set,
		"regex2 matched one of " +
		gostring(response.Header["Set-Cookie"]),
	)
}

// TestDoubleCookiemaskBottomErrorTopNoMasking verifies that a chain of two
// Falcore RequestFilters each generated by NewCookiemaskFilter behaves as
// expected when given a request that contains one cookie that is not to be
// masked and a second cookie that is associated with the second RequestFilter,
// though that RequestFilter will throw an error. Specifically, this function
// verifies that the expected error occurred, that the submitted cookie which
// was not supposed to be masked did in fact make it to the onError
// RequestFilter, and that a new maskable cookie of the expected variant for the
// first RequestFilter in the chain is being set.
func TestDoubleCookiemaskBottomErrorTopNoMasking(t *testing.T) {
	/* setup */
	cookieRegex1 := regexp.MustCompile(
		"^example-test1-[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieNameRandSize*2) +
		"}=[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieValueRandSize*2) +
		"}",
	)
	cookieRegex2 := regexp.MustCompile(
		"^example-error-[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieNameRandSize*2) +
		"}=[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieValueRandSize*2) +
		"}",
	)
	request, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)
	cookie1 := exampleCookieGen("foo")
	cookie2 := exampleCookieGen("error")
	request.AddCookie(&cookie1)
	request.AddCookie(&cookie2)

	/* run */
	_, response := falcore.TestWithRequest(
		request,
		NewCookiemaskFilter(
			exampleCookieMaskGen("test1"),
			NewCookiemaskFilter(
				exampleCookieMaskGen("error"),
				cookieMaskTestPage,
				errorPage,
			),
			errorPage,
		),
		nil,
	)

	/* check */
	assert.Equal(t, 500, response.StatusCode)

	contents, err := ioutil.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.False(t, strings.Contains(string(contents), cookie1.String()))
	assert.False(t, strings.Contains(string(contents), cookie2.String()))
	assert.False(
		t,
		strings.Contains(
			string(contents),
			"example-test1-cookie-found",
		),
		"contents are " + string(contents),
	)
	assert.False(
		t,
		strings.Contains(
			string(contents),
			"example-error-cookie-found",
		),
		"contents are " + string(contents),
	)

	new_cookie1_set := false
	new_cookie2_set := false
	for _, cke_str := range response.Header["Set-Cookie"] {
		if cookieRegex1.MatchString(cke_str) {
			new_cookie1_set = true
		} else if cookieRegex2.MatchString(cke_str) {
			new_cookie2_set = true
		}
	}
	assert.True(
		t,
		new_cookie1_set,
		"regex1 didn't match any of " +
		gostring(response.Header["Set-Cookie"]),
	)
	assert.False(
		t,
		new_cookie2_set,
		"regex2 matched one of " +
		gostring(response.Header["Set-Cookie"]),
	)
}

// TestDoubleCookiemaskBottomErrorTopMasking verifies that a chain of two
// Falcore RequestFilters each generated by NewCookiemaskFilter behaves as
// expected when given a request that contains one cookie that is to be masked
// by the first RequestFilter and a second cookie that is associated with the
// second RequestFilter, though that RequestFilter will throw an error.
// Specifically, this function verifies that the expected error occurred, that
// the submitted cookie which was supposed to be masked did not in fact make it
// to the onError RequestFilter, and that no new maskable cookie is being set.
func TestDoubleCookiemaskBottomErrorTopMasking(t *testing.T) {
	/* setup */
	cookieRegex1 := regexp.MustCompile(
		"^example-test1-[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieNameRandSize*2) +
		"}=[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieValueRandSize*2) +
		"}",
	)
	cookieRegex2 := regexp.MustCompile(
		"^example-error-[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieNameRandSize*2) +
		"}=[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieValueRandSize*2) +
		"}",
	)
	request, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)
	cookie1 := exampleCookieGen("test1")
	cookie2 := exampleCookieGen("error")
	request.AddCookie(&cookie1)
	request.AddCookie(&cookie2)

	/* run */
	_, response := falcore.TestWithRequest(
		request,
		NewCookiemaskFilter(
			exampleCookieMaskGen("test1"),
			NewCookiemaskFilter(
				exampleCookieMaskGen("error"),
				cookieMaskTestPage,
				errorPage,
			),
			errorPage,
		),
		nil,
	)

	/* check */
	assert.Equal(t, 500, response.StatusCode)

	contents, err := ioutil.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.False(t, strings.Contains(string(contents), cookie1.String()))
	assert.False(t, strings.Contains(string(contents), cookie2.String()))
	// BUG(proidiot) issue-#45: context should have been preserved
	assert.False(
		t,
		strings.Contains(
			string(contents),
			"example-test1-cookie-found",
		),
		"contents are " + string(contents),
	)
	assert.False(
		t,
		strings.Contains(
			string(contents),
			"example-error-cookie-found",
		),
		"contents are " + string(contents),
	)

	new_cookie1_set := false
	new_cookie2_set := false
	for _, cke_str := range response.Header["Set-Cookie"] {
		if cookieRegex1.MatchString(cke_str) {
			new_cookie1_set = true
		} else if cookieRegex2.MatchString(cke_str) {
			new_cookie2_set = true
		}
	}
	assert.False(
		t,
		new_cookie1_set,
		"regex1 matched one of " +
		gostring(response.Header["Set-Cookie"]),
	)
	assert.False(
		t,
		new_cookie2_set,
		"regex2 matched one of " +
		gostring(response.Header["Set-Cookie"]),
	)
}

// TestDoubleCookiemaskBottomErrorTopMasking verifies that a chain of two
// Falcore RequestFilters each generated by NewCookiemaskFilter behaves as
// expected when given a request that contains one cookie that is not to be
// masked and a second cookie that is associated with the first RequestFilter,
// though that RequestFilter will throw an error. Specifically, this function
// verifies that the expected error occurred, that the submitted cookie which
// was not supposed to be masked did in fact make it to the onError
// RequestFilter, and that no new maskable cookie is being set.
func TestDoubleCookiemaskTopErrorBottomNoMasking(t *testing.T) {
	/* setup */
	cookieRegex1 := regexp.MustCompile(
		"^example-error-[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieNameRandSize*2) +
		"}=[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieValueRandSize*2) +
		"}",
	)
	cookieRegex2 := regexp.MustCompile(
		"^example-test2-[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieNameRandSize*2) +
		"}=[0-9A-Fa-f]{" +
		strconv.Itoa(cookieMaskTestCookieValueRandSize*2) +
		"}",
	)
	request, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)
	cookie1 := exampleCookieGen("error")
	cookie2 := exampleCookieGen("foo")
	request.AddCookie(&cookie1)
	request.AddCookie(&cookie2)

	/* run */
	_, response := falcore.TestWithRequest(
		request,
		NewCookiemaskFilter(
			exampleCookieMaskGen("error"),
			NewCookiemaskFilter(
				exampleCookieMaskGen("test2"),
				cookieMaskTestPage,
				errorPage,
			),
			errorPage,
		),
		nil,
	)

	/* check */
	assert.Equal(t, 500, response.StatusCode)

	contents, err := ioutil.ReadAll(response.Body)
	assert.NoError(t, err)
	assert.False(t, strings.Contains(string(contents), cookie1.String()))
	assert.False(t, strings.Contains(string(contents), cookie2.String()))
	assert.False(
		t,
		strings.Contains(
			string(contents),
			"example-test1-cookie-found",
		),
		"contents are " + string(contents),
	)
	assert.False(
		t,
		strings.Contains(
			string(contents),
			"example-error-cookie-found",
		),
		"contents are " + string(contents),
	)

	new_cookie1_set := false
	new_cookie2_set := false
	for _, cke_str := range response.Header["Set-Cookie"] {
		if cookieRegex1.MatchString(cke_str) {
			new_cookie1_set = true
		} else if cookieRegex2.MatchString(cke_str) {
			new_cookie2_set = true
		}
	}
	assert.False(
		t,
		new_cookie1_set,
		"regex1 matched one of " +
		gostring(response.Header["Set-Cookie"]),
	)
	assert.False(
		t,
		new_cookie2_set,
		"regex2 matched one of " +
		gostring(response.Header["Set-Cookie"]),
	)
}

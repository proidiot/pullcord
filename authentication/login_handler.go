package authentication

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"github.com/fitstar/falcore"
	// "github.com/stuphlabs/pullcord"
	"net/http"
)

const XsrfTokenLength = 64

type LoginHandler struct {
	Identifier string
	PasswordChecker PasswordChecker
	Downstream falcore.RequestFilter
}

func internalServerError(request *falcore.Request) *http.Response {
	return falcore.StringResponse(
		request.HttpRequest,
		500,
		nil,
		"<html><body><h1>Internal Server Error</h1>" +
		"<p>An internal server error occured." +
		"Please contact your system administrator.</p></body></html>",
	)
}

func notImplementedError(request *falcore.Request) *http.Response {
	return falcore.StringResponse(
		request.HttpRequest,
		501,
		nil,
		"<html><body><h1>Not Implemented</h1>" +
		"<p>The requested behavior has not yet been implemented." +
		"Please contact your system administrator.</p></body></html>",
	)
}

func loginPage(
	request *falcore.Request,
	sesh Session,
	badCreds bool,
) *http.Response {
	return falcore.StringResponse(
		request.HttpRequest,
		501,
		nil,
		"<html><body><h1>Not Implemented</h1>" +
		"<p>The requested behavior has not yet been implemented." +
		"Please contact your system administrator.</p></body></html>",
	)
}

func (handler *LoginHandler) handleLogin(
	request *falcore.Request,
) *http.Response {
	errString := ""
	rawsesh, present := request.Context["session"]
	if !present {
		log().Crit(
			"login handler was unable to retrieve session from" +
			" context",
		)
		return internalServerError(request)
	}
	sesh := rawsesh.(Session)

	authSeshKey := "authenticated-" + handler.Identifier
	xsrfKey := "xsrf-" + handler.Identifier
	usernameKey := "username-" + handler.Identifier
	passwordKey := "password-" + handler.Identifier

	authd, err := sesh.GetValue(authSeshKey)
	if err == nil && authd == true {
		log().Debug("login handler passing request along")
		return handler.Downstream.FilterRequest(request)
	} else if err != NoSuchSessionValueError {
		log().Err(
			fmt.Sprintf(
				"login handler error during auth status" +
				" retrieval: %v",
				err,
			),
		)
		return internalServerError(request)
	}

	xsrfStored, err := sesh.GetValue(xsrfKey)
	if err != nil && err != NoSuchSessionValueError {
		log().Err(
			fmt.Sprintf(
				"login handler error during xsrf token" +
				" retrieval: %v",
				err,
			),
		)
		return internalServerError(request)
	} else if err == NoSuchSessionValueError {
		log().Info("login handler received new request")
	} else if err = request.HttpRequest.ParseForm(); err != nil {
		log().Warning(
			fmt.Sprintf(
				"login handler error during ParseForm: %v",
				err,
			),
		)
		errString = "Bad request"
	} else if xsrfRcvd, present :=
		request.HttpRequest.PostForm[xsrfKey]; ! present {
		log().Info("login handler did not receive xsrf token")
		errString = "Invalid credentials"
	} else  if len(xsrfRcvd) != 1 || 1 != subtle.ConstantTimeCompare(
		[]byte(xsrfStored.(string)),
		[]byte(xsrfRcvd[0]),
	) {
		log().Info("login handler received bad xsrf token")
		errString = "Invalid credentials"
	} else if uVals, present :=
		request.HttpRequest.PostForm[usernameKey]; ! present {
		log().Info("login handler did not receive username")
		errString = "Invalid credentials"
	} else if pVals, present :=
		request.HttpRequest.PostForm[passwordKey]; ! present {
		log().Info("login handler did not receive password")
		errString = "Invalid credentials"
	} else if len(uVals) != 1 || len(pVals) != 1 {
		log().Info(
			"login handler received multi values for username or" +
			" password",
		)
		errString = "Bad request"
	} else if err = handler.PasswordChecker.CheckPassword(
		uVals[0],
		pVals[0],
	); err == NoSuchIdentifierError {
		log().Info("login handler received bad username")
		errString = "Invalid credentials"
	} else if err == BadPasswordError {
		log().Info("login handler received bad password")
		errString = "Invalid credentials"
	} else if err != nil{
		log().Err(
			fmt.Sprintf(
				"login handler error during CheckPassword: %v",
				err,
			),
		)
		return internalServerError(request)
	} else if err = sesh.SetValue(authSeshKey, true); err != nil {
		log().Err(
			fmt.Sprintf(
				"login handler error during auth set: %v",
				err,
			),
		)
		return internalServerError(request)
	} else {
		log().Notice(
			fmt.Sprintf(
				"login successful for: %s",
				uVals[0],
			),
		)
		return handler.Downstream.FilterRequest(request)
	}

	rawXsrfToken := make([]byte, XsrfTokenLength)
	if rsize, err := rand.Read(
		rawXsrfToken[:],
	); err != nil || rsize != XsrfTokenLength {
		log().Err(
			fmt.Sprintf(
				"login handler error during xsrf generation:" +
				" len expected: %u, actual: %u, err: %v",
				XsrfTokenLength,
				rsize,
				err,
			),
		)
		return internalServerError(request)
	}
	nextXsrfToken := hex.EncodeToString(rawXsrfToken)

	if err = sesh.SetValue(xsrfKey, nextXsrfToken); err != nil {
		log().Err(
			fmt.Sprintf(
				"login handler error during xsrf set: %v",
				err,
			),
		)
		return internalServerError(request)
	}

	htr := request.HttpRequest
	htr.ParseForm()
	log().Debug(
		fmt.Sprintf(
			"login handler got form data: %v",
			htr.Form,
		),
	)

	errMarkup := ""
	if errString != "" {
		errMarkup = fmt.Sprintf(
			"<label class=\"error\">%s</label>",
			errString,
		)
	}

	return falcore.StringResponse(
		request.HttpRequest,
		200,
		nil,
		fmt.Sprintf(
			"<html><head><title>Pullcord Login</title></head>" +
			"<body><form method=\"POST\" action=\"%s\">" +
			"<fieldset><legend>Pullcord Login</legend>%s<label " +
			"for=\"username\">Username:</label><input " +
			"type=\"text\" name=\"%s\" id=\"username\" /><label " +
			"for=\"password\">Password:</label><input " +
			"type=\"password\" name=\"%s\" id=\"password\" />" +
			"<input type=\"hidden\" name=\"%s\" value=\"%s\" />" +
			"<input type=\"submit\" value=\"Login\"/></fieldset>" +
			"</form></body></html>",
			request.HttpRequest.URL.Path,
			errMarkup,
			usernameKey,
			passwordKey,
			xsrfKey,
			nextXsrfToken,
		),
	)
}

func NewLoginFilter(
	sessionHandler SessionHandler,
	loginHandler LoginHandler,
) falcore.RequestFilter {
	log().Info("initializing login handler")

	return NewCookiemaskFilter(
		sessionHandler,
		falcore.NewRequestFilter(loginHandler.handleLogin),
		falcore.NewRequestFilter(
			func (request *falcore.Request) *http.Response {
				return internalServerError(request)
			},
		),
	)
}


package authentication

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/proidiot/gone/errors"
	"github.com/proidiot/gone/log"
	"github.com/stuphlabs/pullcord/config"
)

const minSessionCookieNameRandSize = 32
const minSessionCookieValueRandSize = 128
const minSessionCookieMaxAge = 2 * 60 * 60

const rngCollisionError = errors.New(
	"The random number generator produced a colliding value. This is" +
		" perfectly fine if it occurs extremely rarely, but if it" +
		" occurs more than once, it would be extremely concerning.",
)

// MinSessionHandler is a somewhat minimalist form of a SessionHandler.
type MinSessionHandler struct {
	Name   string
	Path   string
	Domain string
	table  map[string]*MinSession
}

func init() {
	config.RegisterResourceType(
		"minsessionhandler",
		func() json.Unmarshaler {
			return new(MinSessionHandler)
		},
	)
}

// UnmarshalJSON implements encoding/json.Unmarshaler.
func (h *MinSessionHandler) UnmarshalJSON(data []byte) error {
	log.Debug("Unmarshaling a MinSessionHandler")
	var t struct {
		Name   string
		Path   string
		Domain string
	}

	if e := json.Unmarshal(data, &t); e != nil {
		return e
	}

	h.Name = t.Name
	h.Path = t.Path
	h.Domain = t.Domain

	return nil
}

// NewMinSessionHandler creates an initialized session handler. Unfortunately a
// nil MinSessionHandler does not currently provide the desired behavior.
func NewMinSessionHandler(name, path, domain string) *MinSessionHandler {
	return &MinSessionHandler{
		name,
		path,
		domain,
		make(map[string]*MinSession),
	}
}

func (handler *MinSessionHandler) assureTableInitialized() {
	if handler.table == nil {
		handler.table = make(map[string]*MinSession)
	}
}

// GetSession generates a new session to track state with.
func (handler *MinSessionHandler) GetSession() (Session, error) {
	log.Info("minsessionhandler is generating a new session")
	log.Debug(
		fmt.Sprintf(
			"minsession being generated by minsessionhandler: %v",
			handler,
		),
	)

	handler.assureTableInitialized()

	var sesh MinSession
	var core minSessionCore
	sesh.core = &core
	sesh.core.data = make(map[string]interface{})
	sesh.handler = handler
	return &sesh, nil
}

func (handler *MinSessionHandler) genCookie() (*http.Cookie, error) {
	log.Debug("minsession generating cookie")

	handler.assureTableInitialized()

	var randReadErr error
	var otherErr error

	nbytes := make([]byte, minSessionCookieNameRandSize)
	vbytes := make([]byte, minSessionCookieValueRandSize)

	cookieName := ""
	nameGenNeeded := true
	for nameGenNeeded {
		_, randReadErr = rand.Read(nbytes)
		if randReadErr != nil {
			log.Err(
				"minsession cookie generation was unable to" +
					" read the needed random bytes",
			)
			return nil, randReadErr
		}

		cookieName = handler.Name + "-" + hex.EncodeToString(nbytes)
		// Slower than adding an else on the next if, but clearer.
		nameGenNeeded = false

		_, collision := handler.table[cookieName]
		if collision {
			// We had a collision with an existing legit cookie?
			// Is the random number generator broken?
			log.Err(
				fmt.Sprintf(
					"minsession cookie generation has"+
						" created a new cookie with a"+
						" name that collides with an"+
						" already existing cookie from"+
						" the cookie table which"+
						" shares the name: %s",
					cookieName,
				),
			)
			nameGenNeeded = true
			otherErr = rngCollisionError
		}
	}

	_, randReadErr = rand.Read(vbytes)
	if randReadErr != nil {
		log.Err(
			"minsession cookie generation was unable to" +
				" read the needed random bytes",
		)
		return nil, randReadErr
	}

	var cke http.Cookie
	cke.Name = cookieName
	cke.Value = hex.EncodeToString(vbytes)
	cke.Path = handler.Path
	cke.Domain = handler.Domain
	cke.MaxAge = minSessionCookieMaxAge
	// TODO make configurable
	cke.Secure = false
	cke.HttpOnly = true

	return &cke, otherErr
}

type minSessionCore struct {
	cvalue string
	data   map[string]interface{}
}

// MinSession represents a particular user session, and is intrinsically linked
// with the MinSessionHandler that created it.
type MinSession struct {
	core    *minSessionCore
	handler *MinSessionHandler
}

// GetValue looks up a value stored in the session for a given key. If the key
// did not match an existing saved value, an error will be returned.
func (sesh *MinSession) GetValue(key string) (interface{}, error) {
	log.Debug(fmt.Sprintf("minsession requesting value for: %s", key))

	value, present := sesh.core.data[key]
	if !present {
		return nil, NoSuchSessionValueError
	}

	return value, nil
}

// GetValues retrieves all the key/value pairs associated with the session.
func (sesh *MinSession) GetValues() map[string]interface{} {
	log.Debug("minsession requesting all values")

	return sesh.core.data
}

// SetValue saves a key/value pair into the session, possibly overwriting a
// previously set value.
func (sesh *MinSession) SetValue(key string, value interface{}) error {
	log.Debug(fmt.Sprintf("minsession setting value for: %s", key))

	sesh.core.data[key] = value
	return nil
}

// CookieMask for the MinSession is an implementation of the CookieMask
// function required by all Session derivatives.
func (sesh *MinSession) CookieMask(incomingCookies []*http.Cookie) (
	forwardedCookies []*http.Cookie,
	setCookies []*http.Cookie,
	err error,
) {
	log.Debug("running minsession cookiemask")

	newCookieNeeded := true
	cookieNameRegex := regexp.MustCompile(
		"^" +
			sesh.handler.Name +
			"-[0-9A-Fa-f]{" +
			strconv.Itoa(minSessionCookieNameRandSize*2) +
			"}$",
	)

	inCkesBuffer := new(bytes.Buffer)
	for _, cookie := range incomingCookies {
		inCkesBuffer.WriteString("\"" + cookie.Name + "\",")

		if cookieNameRegex.MatchString(cookie.Name) {
			sesh2, present := sesh.handler.table[cookie.Name]
			if present &&
				len(cookie.Value) > 0 &&
				cookie.Value == sesh2.core.cvalue {
				log.Debug(
					fmt.Sprintf(
						"minsession cookiemask"+
							" received valid"+
							" cookie with name: %s",
						cookie.Name,
					),
				)

				newCookieNeeded = false

				// In the more common scenario where a new
				// empty session was created, it would probably
				// be faster to copy any recently set values
				// into the already established session than
				// vice-versa.
				for key, val := range sesh.core.data {
					sesh2.core.data[key] = val
				}

				// This should replace a new empty session with
				// an already established one, but I think it
				// would work as expected with two already
				// established sessions as well? I hope so, at
				// least... Anyway, we can't just exit here in
				// case such a scenario occurs.
				sesh.core = sesh2.core
			} else {
				if present {
					// TODO: configurable info vs warn?
					log.Info(
						fmt.Sprintf(
							"minsession"+
								" cookiemask"+
								" received bad"+
								" cookie value"+
								" for valid"+
								" cookie name:"+
								" %s",
							cookie.Name,
						),
					)

					delete(sesh.handler.table, cookie.Name)

					// TODO: should this destroy every
					// session that is touched?
				} else {
					log.Info(
						fmt.Sprintf(
							"minsession"+
								" cookiemask"+
								" received"+
								" matching but"+
								" invalid"+
								" cookie with"+
								" name: %s",
							cookie.Name,
						),
					)
				}

				var badCookie http.Cookie
				badCookie.Name = cookie.Name
				badCookie.Value = cookie.Value
				badCookie.MaxAge = -1
				setCookies = append(setCookies, &badCookie)
			}
		} else {
			forwardedCookies = append(forwardedCookies, cookie)
		}
	}
	log.Debug(
		fmt.Sprintf(
			"minsession cookiemask received cookies with these"+
				" cookie names: [%s]",
			inCkesBuffer.String(),
		),
	)

	if newCookieNeeded {
		log.Debug(
			"minsession cookiemask needs to generate a new cookie",
		)

		newCookie, err := sesh.handler.genCookie()
		if newCookie == nil {
			log.Err(
				fmt.Sprintf(
					"minsession cookiemask ran into"+
						" unexpected, unrecoverable"+
						" errors during cookie"+
						" generation: %s",
					err,
				),
			)
		} else if err != nil && err != rngCollisionError {
			log.Warning(
				fmt.Sprintf(
					"minsession cookiemask ran into"+
						" unexpected but apparently"+
						" recoverable errors during"+
						" cookie generation: %s",
					err,
				),
			)
		}

		setCookies = append(setCookies, newCookie)
		log.Info(
			fmt.Sprintf(
				"minsession cookiemask has added a new cookie"+
					" with name: %s",
				newCookie.Name,
			),
		)

		sesh.core.cvalue = newCookie.Value
		sesh.handler.table[newCookie.Name] = sesh
		log.Debug(
			fmt.Sprintf(
				"minsession cookiemask has created a new"+
					" session to go with the new cookie"+
					" with name: %s",
				newCookie.Name,
			),
		)
	}

	return forwardedCookies, setCookies, nil
}

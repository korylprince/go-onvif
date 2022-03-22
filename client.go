// package onvif aims to be a simple to use, idiomatic ONVIF client library.
package onvif

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/icholy/digest"
	"github.com/korylprince/go-onvif/soap"
)

// AuthMode represents an ONVIF request authentication mode. See Client.AuthMode for more information
type AuthMode int

// ONVIF authentication modes
const (
	AuthModeNone AuthMode = iota
	AuthModeDigest
	AuthModeWSSecurity
)

// Request is a SOAP request
type Request struct {
	URL string
	// Namespaces will be added to the SOAP envelope
	Namespaces soap.Namespaces
	// Body will be marshaled to XML as the SOAP body contents
	Body interface{}
}

// Client is an ONVIF client
type Client struct {
	// AuthMode specifies which authentication mode to use to authenticate requests.
	// If set to AuthModeNone (the default value), the Client will not use authentication unless an authorization error occurs.
	// In that case, if Username and Password are set, the Client will attempt to detect the correct AuthMode,
	// update Client.AuthMode, and authenticate all future requests.
	// If AuthMode is set to AuthModeWSSecurity and an HTTP 401 response is returned (indicated WS Security tokens are not supported),
	// AuthMode will be set to AuthModeDigest.
	AuthMode
	Username string
	Password string
	// HTTPClient is the *http.Client to use for the request. If nil, http.DefaultClient is used
	HTTPClient *http.Client
	// If Debug is true, the client will print the full request and response to stdout
	Debug bool
}

type fakeTransport struct {
	resp *http.Response
}

func (f *fakeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f.resp, nil
}

// Do executes a SOAP request.
// The response envelope is returned, which can be further unmarshaled with soap.Body.Unmarshal
// If the device returns a *soap.Fault, it will be returned as an error
func (c *Client) Do(r *Request) (*soap.Envelope, error) {
	// set default Client
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{}
	}

	var (
		s   *soap.Security
		err error
	)

	// set auth params
	if c.Username != "" && c.Password != "" {
		switch c.AuthMode {
		case AuthModeNone:
		case AuthModeWSSecurity:
			s, err = soap.NewSecurity(c.Username, c.Password)
			if err != nil {
				return nil, fmt.Errorf("could not create security header: %w", err)
			}
		case AuthModeDigest:
			if _, ok := c.HTTPClient.Transport.(*digest.Transport); !ok {
				c.HTTPClient.Transport = &digest.Transport{Username: c.Username, Password: c.Password}
			}
		default:
			return nil, fmt.Errorf("invalid SecurityType: %d", c.AuthMode)
		}
	}

	// marshal request
	buf, err := xml.Marshal(r.Body)
	if err != nil {
		return nil, fmt.Errorf("could not marshal request: %w", err)
	}

	env := &soap.Envelope{
		Namespaces: r.Namespaces,
		Header: &soap.Header{
			Security: s,
		},
		Body: &soap.Body{InnerXML: buf},
	}

	buf2 := bytes.NewBufferString(xml.Header)

	if err = xml.NewEncoder(buf2).Encode(env); err != nil {
		return nil, fmt.Errorf("could not marshal envelope: %w", err)
	}

	if c.Debug {
		fmt.Printf("Request:\n%s\n", buf2.String())
	}

	// create http request
	httpReq, err := http.NewRequest(http.MethodPost, r.URL, buf2)
	if err != nil {
		return nil, fmt.Errorf("could not create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/soap+xml")

	// send request
	soapResp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("could not POST request: %w", err)
	}
	defer soapResp.Body.Close()

	if c.Debug && soapResp != nil {
		buf2 = new(bytes.Buffer)
		if _, err := buf2.ReadFrom(soapResp.Body); err != nil {
			return nil, fmt.Errorf("could not read response body: %w", err)
		}
		fmt.Printf("Response:\n%s\n", buf2.String())
		soapResp.Body = io.NopCloser(buf2)
	}

	// check for digest auth error
	if soapResp.StatusCode == http.StatusUnauthorized {
		if c.AuthMode != AuthModeDigest && c.Username != "" && c.Password != "" {
			c.AuthMode = AuthModeDigest
			if _, ok := c.HTTPClient.Transport.(*digest.Transport); !ok {
				// replay request to save digest headers
				d := &digest.Transport{Transport: &fakeTransport{resp: soapResp}, Username: c.Username, Password: c.Password}
				d.RoundTrip(httpReq)
				d.Transport = nil
				c.HTTPClient.Transport = d
			}
			return c.Do(r)
		}
		return nil, &soap.UnauthorizedError{Err: errors.New(soapResp.Status)}
	}

	// parse response
	env = new(soap.Envelope)
	if err = xml.NewDecoder(soapResp.Body).Decode(env); err != nil {
		return nil, fmt.Errorf("could not decode response: %w", err)
	}

	// check for soap fault
	if env.Body.Fault != nil {
		if env.Body.Fault.IsUnauthorizedError() {
			if c.AuthMode == AuthModeNone && c.Username != "" && c.Password != "" {
				c.AuthMode = AuthModeWSSecurity
				return c.Do(r)
			}
			return nil, &soap.UnauthorizedError{Err: env.Body.Fault}
		}
		return nil, env.Body.Fault
	}

	return env, nil
}

// package onvif aims to be a simple to use, idiomatic ONVIF client library.
package onvif

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/korylprince/go-onvif/soap"
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
	// If Username and Password are set, they'll be used to authenticate the request
	Username string
	Password string
	// HTTPClient is the *http.Client to use for the request. If nil, http.DefaultClient is used
	HTTPClient *http.Client
	// If Debug is true, the client will print the full request and response to stdout
	Debug bool
}

// Do executes a SOAP request.
// The response envelope is returned, which can be further unmarshaled with soap.Body.Unmarshal
// If the device returns a *soap.Fault, it will be returned as an error
func (c *Client) Do(r *Request) (*soap.Envelope, error) {
	var (
		s   *soap.Security
		err error
	)
	if c.Username != "" && c.Password != "" {
		s, err = soap.NewSecurity(c.Username, c.Password)
		if err != nil {
			return nil, fmt.Errorf("could not create security header: %w", err)
		}
	}

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

	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	soapResp, err := client.Post(r.URL, "application/soap+xml", buf2)
	if err != nil {
		return nil, fmt.Errorf("could not POST request: %w", err)
	}
	defer soapResp.Body.Close()

	if c.Debug {
		buf2 = new(bytes.Buffer)
		if _, err := buf2.ReadFrom(soapResp.Body); err != nil {
			return nil, fmt.Errorf("could not read response body: %w", err)
		}
		fmt.Printf("Response:\n%s\n", buf2.String())
		soapResp.Body = io.NopCloser(buf2)
	}

	env = new(soap.Envelope)
	if err = xml.NewDecoder(soapResp.Body).Decode(env); err != nil {
		// if body can't be parsed, make sure device didn't send HTTP 401
		if soapResp.StatusCode == http.StatusUnauthorized {
			return nil, &soap.UnauthorizedError{Err: errors.New(soapResp.Status)}
		}
		return nil, fmt.Errorf("could not decode response: %w", err)
	}

	if env.Body.Fault != nil {
		if strings.ToLower(env.Body.Fault.Reason) == "sender not authorized" {
			return nil, &soap.UnauthorizedError{Err: env.Body.Fault}
		}
		return nil, env.Body.Fault
	}

	return env, nil
}

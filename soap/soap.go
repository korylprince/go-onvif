package soap

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
)

// SOAP Namespaces
const (
	NamespaceEnvelope = "http://www.w3.org/2003/05/soap-envelope"

	NamspaceWSSSecExt   = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd"
	NamespaceWSSUtility = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd"

	NamespaceONVIFError = "http://www.onvif.org/ver10/error"
)

// ErrNoResponse indicates a SOAP response was not returned
var ErrNoResponse = errors.New("server did not return a response")

// UnexpectedTokenError is an unexpected token
type UnexpectedTokenError xml.Name

func (e UnexpectedTokenError) Error() string {
	return fmt.Sprintf("unexpected token: %v", xml.Name(e))
}

// UnexpectedTokenTypeError is an unexpected xml.Token type during unmarshaling
type UnexpectedTokenTypeError struct {
	Token xml.Token
}

func (e UnexpectedTokenTypeError) Error() string {
	return fmt.Sprintf("unexpected token type: %T", e.Token)
}

// UnauthorizedError indicates invalid credentials
type UnauthorizedError struct {
	Err error
}

func (e *UnauthorizedError) Error() string {
	return fmt.Sprintf("unauthorized: %s", e.Err.Error())
}

// Namespaces is a mapping of XML namespaces of the form xmlns:<name> -> <url>
//
// Example: Namespaces{"tds": "http://www.onvif.org/ver10/device/wsdl"}
type Namespaces map[string]string

// Envelope is the body of the SOAP message
type Envelope struct {
	// Namespaces is the additional namespaces set on the envelope
	Namespaces map[string]string `xml:"-"`
	Header     *Header
	// Body is guaranteed to be non-nil when the envelope is unmarshaled from XML
	Body *Body
}

// MarshalXML implements xml.Marshaler
func (e *Envelope) MarshalXML(enc *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "env:Envelope"}

	start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "xmlns:env"}, Value: NamespaceEnvelope})

	for name, val := range e.Namespaces {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "xmlns:" + name}, Value: val})
	}

	if err := enc.EncodeToken(start); err != nil {
		return fmt.Errorf("could not encode start token: %w", err)
	}

	if e.Header != nil {
		h := &header{Security: e.Header.Security}
		if err := enc.Encode(h); err != nil {
			return fmt.Errorf("could not encode header: %w", err)
		}
	}

	if e.Body != nil {
		b := &body{Fault: e.Body.Fault, InnerXML: e.Body.InnerXML}
		if err := enc.Encode(b); err != nil {
			return fmt.Errorf("could not encode body: %w", err)
		}
	}

	if err := enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "env:Envelope"}}); err != nil {
		return fmt.Errorf("could not encode end token: %w", err)
	}

	return nil
}

// UnmarshalXML implements xml.Unmarshaler
func (e *Envelope) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if e.Namespaces == nil {
		e.Namespaces = make(Namespaces)
	}
	for _, attr := range start.Attr {
		if strings.ToLower(attr.Name.Space) == "xmlns" {
			e.Namespaces[attr.Name.Local] = attr.Value
		}
	}

	// guarantee body is not nil
	if e.Body == nil {
		e.Body = new(Body)
	}

	for {
		tok, err := d.Token()
		if err != nil {
			return fmt.Errorf("could not decode token: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "Header" {
				h := new(Header)
				if err = d.DecodeElement(h, &t); err != nil {
					return fmt.Errorf("could not decode header: %w", err)
				}
				e.Header = h
			} else if t.Name.Local == "Body" {
				b := new(Body)
				if err = d.DecodeElement(b, &t); err != nil {
					return fmt.Errorf("could not decode body: %w", err)
				}
				e.Body = b
				if e.Body.Fault != nil {
					e.Body.Fault.Namespaces = e.Namespaces
				}
			} else {
				return UnexpectedTokenError(t.Name)
			}
		case xml.EndElement:
			return nil
		case xml.CharData:
			if len(bytes.TrimSpace(t)) != 0 {
				return UnexpectedTokenTypeError{Token: tok}
			}
		default:
			return UnexpectedTokenTypeError{Token: tok}
		}
	}
}

// Header is a SOAP message header
type Header struct {
	XMLName  xml.Name  `xml:"Header"`
	Security *Security `xml:",omitempty"`
}

type header struct {
	XMLName  xml.Name  `xml:"env:Header"`
	Security *Security `xml:",omitempty"`
}

// Fault is a SOAP message error
type Fault struct {
	Namespaces map[string]string `xml:"-"`
	XMLName    xml.Name          `xml:"Fault"`
	Code       string            `xml:"Code>Value"`
	SubCode    string            `xml:"Code>Subcode>Value"`
	Reason     string            `xml:"Reason>Text"`
	Node       string
	Role       string
	Detail     []byte `xml:",innerxml"`
}

func (f *Fault) Error() string {
	var codes []string
	if f.Code != "" {
		codes = append(codes, f.Code)
	}
	if f.SubCode != "" {
		codes = append(codes, f.SubCode)
	}
	c := ""
	if len(codes) > 0 {
		c = fmt.Sprintf(" (%s)", strings.Join(codes, ", "))
	}

	return fmt.Sprintf("SOAP fault%s: %s", c, f.Reason)
}

// IsUnauthorizedError returns true if the fault indicates an authorization error
func (f *Fault) IsUnauthorizedError() bool {
	soapPrefix := ""
	errPrefix := ""
	for prefix, ns := range f.Namespaces {
		if ns == NamespaceEnvelope {
			soapPrefix = prefix + ":"
		} else if ns == NamespaceONVIFError {
			errPrefix = prefix + ":"
		}
	}
	return f.Code == soapPrefix+"Sender" && f.SubCode == errPrefix+"NotAuthorized"
}

// Body is a SOAP message body
type Body struct {
	XMLName  xml.Name `xml:"Body"`
	Fault    *Fault   `xml:",omitempty"`
	InnerXML []byte   `xml:",innerxml"`
}

// Unmarshal unmarshals the envelope body into v
func (b *Body) Unmarshal(v interface{}) error {
	if len(b.InnerXML) == 0 {
		return fmt.Errorf("body is empty: %w", ErrNoResponse)
	}

	if err := xml.NewDecoder(bytes.NewBuffer(b.InnerXML)).Decode(v); err != nil {
		return fmt.Errorf("could not unmarshal: %w", err)
	}

	return nil
}

type body struct {
	XMLName  xml.Name `xml:"env:Body"`
	Fault    *Fault   `xml:",omitempty"`
	InnerXML []byte   `xml:",innerxml"`
}

package soap

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"time"
)

const (
	typePassword = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordDigest"
	typeNonce    = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-soap-message-security-1.0#Base64Binary"
)

// Security is a SOAP security header
type Security struct {
	UsernameToken *UsernameToken
}

// MarshalXML implements xml.Marshaler
func (s *Security) MarshalXML(enc *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "wsse:Security"}
	start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "xmlns:wsse"}, Value: NamspaceWSSSecExt})
	start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "xmlns:wsu"}, Value: NamespaceWSSUtility})

	if err := enc.EncodeToken(start); err != nil {
		return fmt.Errorf("could not encode start token: %w", err)
	}

	if err := enc.Encode(s.UsernameToken); err != nil {
		return fmt.Errorf("could not encode username token: %w", err)
	}

	if err := enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "wsse:Security"}}); err != nil {
		return fmt.Errorf("could not encode end token: %w", err)
	}

	return nil
}

// Password is the password part of the username token
type Password struct {
	XMLName  xml.Name `xml:"wsse:Password"`
	Type     string   `xml:",attr"`
	Password string   `xml:",chardata"`
}

// Nonce is the nonce part of the username token
type Nonce struct {
	XMLName      xml.Name `xml:"wsse:Nonce"`
	EncodingType string   `xml:",attr"`
	Nonce        string   `xml:",chardata"`
}

// UsernameToken is a SOAP username token
type UsernameToken struct {
	XMLName  xml.Name `xml:"wsse:UsernameToken"`
	Username string   `xml:"wsse:Username"`
	Password *Password
	Nonce    *Nonce
	Created  string `xml:"wsu:Created"`
}

// NewSecurity returns the SOAP Security header
func NewSecurity(username, password string) (*Security, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("could not generate nonce: %w", err)
	}
	created := time.Now().UTC().Format("2006-01-02T15:04:05")

	hash := sha1.New()
	hash.Write(nonce)
	hash.Write([]byte(created))
	hash.Write([]byte(password))

	return &Security{
		UsernameToken: &UsernameToken{
			Username: username,
			Password: &Password{
				Type:     typePassword,
				Password: base64.StdEncoding.EncodeToString(hash.Sum(nil)),
			},
			Nonce: &Nonce{
				EncodingType: typeNonce,
				Nonce:        base64.StdEncoding.EncodeToString(nonce),
			},
			Created: created,
		},
	}, nil
}

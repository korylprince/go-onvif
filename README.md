[![pkg.go.dev](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/korylprince/go-onvif)

# About

When trying to use Go to talk to ONVIF IP cameras, I became frustrated with existing ONVIF or SOAP libraries. Some had poor documentation and some were too highly abstracted to use easily. XML, SOAP, and ONVIF are frustrating enough by themselves, because of the many (incomplete, varying, bad) implementations out there, but ultimately it's just POSTing some XML structures to a service URL.

This library is mostly a small SOAP wrapper built on top of net/http.Client with some ONVIF helper functions. There is as little magic as possible. There's only a few types, and requests and responses are easily inspected (client.Debug).

It's not a project goal to have pre-built types for all the ONVIF types; instead the user should construct the types they need ([gowsdl](https://github.com/hooklift/gowsdl) can be a useful tool to do that).

[pkg.go.dev has a full example](https://pkg.go.dev/github.com/korylprince/go-onvif#pkg-examples).

## API Stability

I don't expect the API to change at this point, but because it's fairly new, it won't be tagged with v1.x.x until the library gets more usage.

# Interacting with an ONVIF device

Generally, you want to interact with an ONVIF device by finding out what services it supports (and what the service URLs are):

```go
c := &onvif.Client{}
services, err := c.GetServices("192.168.0.64:80")
if err != nil {
    // handle err
    panic(err)
}
```

Next you can verify the device has a specific service by checking if it supports that service namespace:

```go
// check if media service is supported
mediaURL := services.URL(onvif.NamespaceMedia)
if mediaURL == "" {
    // handle service not being supported
}

```

Next you can create an `onvif.Request` using the namespace, URL, and an ONVIF request type (see Creating Types section below), and execute it with `onvif.client.Do`. The response body can be Unmarshaled into an ONVIF response type (see Creating Types section below) with `soap.Body.Unmarshal`.

See a full example on [pkg.go.dev](https://pkg.go.dev/github.com/korylprince/go-onvif#pkg-examples).

# Creating Types

If you're not familiar with SOAP/XML, creating Go types to marshal/unmarshal ONVIF types can be frustrating, because you get to deal with XML namespaces and prefixes. There's a few issues with Go's handling of XML namespaces in `encoding/xml` (most of which are outlined [here](https://github.com/ydnar/go/commit/cea873cd245536a7a464d24bf3b24044719daca6)), so we have to be careful of how types are constructed. We'll take a look at `GetCapabilities` and `GetCapabilitiesResponse` as an example:

```go
// GetCapabilities is an ONVIF GetCapabilities operation
type GetCapabilities struct {
    XMLName  xml.Name `xml:"tds:GetCapabilities"`
    Category string   `xml:"tds:Category"`
}
```

You can see that the request type, `GetCapabilities`, has the `tds` prefix in the xml tags. This is because SOAP servers usually expect a prefix on tags. It doesn't matter what the prefix as, as long as it matches the correct namespace in the Request:

```go
req := &Request{
    URL:        deviceURL,
    Namespaces: soap.Namespaces{"tds": onvif.NamespaceDevice},
    Body:       &GetCapabilities{Category: "All"},
}
```

The response type, `GetCapabilitiesResponse` does not have any prefixes:

```go
// GetCapabilitiesResponse is an ONVIF GetServicesResponse response
type GetCapabilitiesResponse struct {
    DeviceURL          string `xml:"Capabilities>Device>XAddr"`
    EventsURL          string `xml:"Capabilities>Events>XAddr"`
    ImagingURL         string `xml:"Capabilities>Imaging>XAddr"`
    MediaURL           string `xml:"Capabilities>Media>XAddr"`
    PTZURL             string `xml:"Capabilities>PTZ>XAddr"`
    DeviceIOURL        string `xml:"Capabilities>Extension>DeviceIO>XAddr"`
    DisplayURL         string `xml:"Capabilities>Extension>Display>XAddr"`
    RecordingURL       string `xml:"Capabilities>Extension>Recording>XAddr"`
    SearchURL          string `xml:"Capabilities>Extension>Search>XAddr"`
    ReplayURL          string `xml:"Capabilities>Extension>Replay>XAddr"`
    ReceiverURL        string `xml:"Capabilities>Extension>Receiver>XAddr"`
    AnalyticsDeviceURL string `xml:"Capabilities>Extension>AnalyticsDevice>XAddr"`
}
```

This is because of the current Go parser issues of matching tags with prefixes. Instead you should just use the tag name by itself.

*Note: the `>` characters in the xml tag are a feature of `xml/encoding` unmarshaling. They allow you to select sub elements at the root struct instead of creating several layers of structs to get to the field. Don't confuse them as being a special syntax needed for ONVIF response types.*

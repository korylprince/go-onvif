package onvif

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/korylprince/go-onvif/soap"
)

// ONVIF Namespaces
const (
	NamespaceDevice = "http://www.onvif.org/ver10/device/wsdl"
	NamespaceEvents = "http://www.onvif.org/ver10/events/wsdl"

	NamespaceAccessControl          = "http://www.onvif.org/ver10/accesscontrol/wsdl"
	NamespaceAccessRules            = "http://www.onvif.org/ver10/accessrules/wsdl"
	NamespaceActionEngine           = "http://www.onvif.org/ver10/actionengine/wsdl"
	NamespaceAdvancedSecurity       = "http://www.onvif.org/ver10/advancedsecurity/wsdl"
	NamespaceAnalytics              = "http://www.onvif.org/ver20/analytics/wsdl"
	NamespaceAnalyticsDevice        = "http://www.onvif.org/ver10/analyticsdevice/wsdl"
	NamespaceAppMgmt                = "http://www.onvif.org/ver10/appmgmt/wsdl"
	NamespaceAuthenticationBehavior = "http://www.onvif.org/ver10/authenticationbehavior/wsdl"
	NamespaceCredential             = "http://www.onvif.org/ver10/credential/wsdl"
	NamespaceDeviceIO               = "http://www.onvif.org/ver10/deviceIO/wsdl"
	NamespaceDisplay                = "http://www.onvif.org/ver10/display/wsdl"
	NamespaceDoorControl            = "http://www.onvif.org/ver10/doorcontrol/wsdl"
	NamespaceFederatedSearch        = "http://www.onvif.org/ver10/federatedsearch/wsdl"
	NamespaceImaging                = "http://www.onvif.org/ver20/imaging/wsdl"
	NamespaceMedia                  = "http://www.onvif.org/ver10/media/wsdl"
	NamespaceMedia2                 = "http://www.onvif.org/ver20/media/wsdl"
	NamespacePTZ                    = "http://www.onvif.org/ver20/ptz/wsdl"
	NamespaceProvisioning           = "http://www.onvif.org/ver10/provisioning/wsdl"
	NamespaceReceiver               = "http://www.onvif.org/ver10/receiver/wsdl"
	NamespaceRecording              = "http://www.onvif.org/ver10/recording/wsdl"
	NamespaceReplay                 = "http://www.onvif.org/ver10/replay/wsdl"
	NamespaceSchedule               = "http://www.onvif.org/ver10/schedule/wsdl"
	NamespaceSearch                 = "http://www.onvif.org/ver10/search/wsdl"
	NamespaceThermal                = "http://www.onvif.org/ver10/thermal/wsdl"
	NamespaceUplink                 = "http://www.onvif.org/ver10/uplink/wsdl"
)

// GetServices is an ONVIF GetServices operation
type GetServices struct {
	XMLName           xml.Name `xml:"tds:GetServices"`
	IncludeCapability bool     `xml:"tds:IncludeCapability"`
}

// Service represents an ONVIF service
type Service struct {
	Namespace    string
	URL          string `xml:"XAddr"`
	VersionMajor int    `xml:"Version>Major"`
	VersionMinor int    `xml:"Version>Minor"`
}

// Services is a list of Services
type Services []*Service

// URL returns the service URL for the given namespace or the empty string if the service isn't found
func (s Services) URL(namespace string) string {
	for _, svc := range s {
		if svc.Namespace == namespace {
			return svc.URL
		}
	}

	return ""
}

// GetServicesResponse is an ONVIF GetServicesResponse response
type GetServicesResponse struct {
	Service Services
}

// GetServices returns the service urls from the remote device.
// addr is the host:port pair of the device. Just the host part can be specified as well.
func (c *Client) GetServices(addr string) (Services, error) {
	req := &Request{
		URL:        fmt.Sprintf("http://%s/onvif/device_service", addr),
		Namespaces: soap.Namespaces{"tds": NamespaceDevice},
		Body:       &GetServices{IncludeCapability: false},
	}
	env, err := c.Do(req)
	if err != nil {
		// if GetServices isn't implemented, try GetCapabilities
		if f, ok := err.(*soap.Fault); ok {
			if strings.Contains(strings.ToLower(f.Reason), "unknown action") || strings.Contains(strings.ToLower(f.Reason), "not implemented") {
				services, err := c.GetCapabilities(addr)
				if err != nil {
					return nil, fmt.Errorf("could not get services via GetServices or GetCapabilities: %w", err)
				}
				return services, nil
			}
		}

		return nil, fmt.Errorf("could not complete operation: %w", err)
	}

	services := new(GetServicesResponse)
	if err := env.Body.Unmarshal(services); err != nil {
		return nil, fmt.Errorf("could not unmarshal response: %w", err)
	}

	return services.Service, nil
}

// GetCapabilities is an ONVIF GetCapabilities operation
type GetCapabilities struct {
	XMLName  xml.Name `xml:"tds:GetCapabilities"`
	Category string   `xml:"tds:Category"`
}

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

// GetCapabilities returns the service urls from the remote device. Most users should use GetServices instead.
// addr is the host:port pair of the device. Just the host part can be specified as well.
func (c *Client) GetCapabilities(addr string) (Services, error) {
	req := &Request{
		URL:        fmt.Sprintf("http://%s/onvif/device_service", addr),
		Namespaces: soap.Namespaces{"tds": NamespaceDevice},
		Body:       &GetCapabilities{Category: "All"},
	}
	env, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not complete operation: %w", err)
	}

	cap := new(GetCapabilitiesResponse)
	if err := env.Body.Unmarshal(cap); err != nil {
		return nil, fmt.Errorf("could not unmarshal response: %w", err)
	}

	var services Services
	if url := cap.DeviceURL; url != "" {
		services = append(services, &Service{Namespace: NamespaceDevice, URL: url})
	}
	if url := cap.EventsURL; url != "" {
		services = append(services, &Service{Namespace: NamespaceEvents, URL: url})
	}
	if url := cap.ImagingURL; url != "" {
		services = append(services, &Service{Namespace: NamespaceImaging, URL: url})
	}
	if url := cap.MediaURL; url != "" {
		services = append(services, &Service{Namespace: NamespaceMedia, URL: url})
	}
	if url := cap.PTZURL; url != "" {
		services = append(services, &Service{Namespace: NamespacePTZ, URL: url})
	}
	if url := cap.DeviceIOURL; url != "" {
		services = append(services, &Service{Namespace: NamespaceDeviceIO, URL: url})
	}
	if url := cap.DisplayURL; url != "" {
		services = append(services, &Service{Namespace: NamespaceDisplay, URL: url})
	}
	if url := cap.RecordingURL; url != "" {
		services = append(services, &Service{Namespace: NamespaceRecording, URL: url})
	}
	if url := cap.SearchURL; url != "" {
		services = append(services, &Service{Namespace: NamespaceSearch, URL: url})
	}
	if url := cap.ReplayURL; url != "" {
		services = append(services, &Service{Namespace: NamespaceReplay, URL: url})
	}
	if url := cap.ReceiverURL; url != "" {
		services = append(services, &Service{Namespace: NamespaceReceiver, URL: url})
	}
	if url := cap.AnalyticsDeviceURL; url != "" {
		services = append(services, &Service{Namespace: NamespaceAnalyticsDevice, URL: url})
	}

	return services, nil
}

package onvif_test

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/korylprince/go-onvif"
	"github.com/korylprince/go-onvif/soap"
)

// GetVideoSources is an ONVIF GetVideoSources operation
type GetVideoSources struct {
	XMLName xml.Name `xml:"trt:GetVideoSources"`
}

//GetVideoSourcesResponse is an ONVIF GetVideoSourcesResponse response
type GetVideoSourcesResponse struct {
	VideoSources []*struct {
		Token     string `xml:"token,attr"`
		Framerate float64
		Width     int `xml:"Resolution>Width"`
		Height    int `xml:"Resolution>Height"`
	}
}

// StringList is an XML StringList type
type StringList []string

func (sl *StringList) UnmarshalText(text []byte) error {
	for _, b := range bytes.Split(text, []byte(" ")) {
		*sl = append(*sl, string(b))
	}
	return nil
}

// GetVideoSourceModes is an ONVIF GetVideoSourceModes operation
type GetVideoSourceModes struct {
	XMLName xml.Name `xml:"trt:GetVideoSourceModes"`
	Token   string   `xml:"trt:VideoSourceToken"`
}

//GetVideoSourceModesResponse is an ONVIF GetVideoSourceModesResponse response
type GetVideoSourceModesResponse struct {
	VideoSourceModes []*struct {
		Token        string `xml:"token,attr"`
		MaxFramerate float64
		MaxWidth     int `xml:"MaxResolution>Width"`
		MaxHeight    int `xml:"MaxResolution>Height"`
		Encodings    StringList
	}
}

// Example of getting all video modes from a camera
func Example() {
	c := &onvif.Client{Username: "admin", Password: "12345"}
	services, err := c.GetServices("192.168.0.64:80")
	if err != nil {
		// handle err
		panic(err)
	}

	var (
		mediaNS  string
		mediaURL string
	)

	// check if media service is supported
	if s := services.URL(onvif.NamespaceMedia); s != "" {
		mediaNS = onvif.NamespaceMedia
		mediaURL = s
	}

	if mediaNS == "" {
		// handle media service not supported
		panic("media service not supported")
	}

	r := &onvif.Request{
		URL:        mediaURL,
		Namespaces: soap.Namespaces{"trt": mediaNS},
		Body:       &GetVideoSources{},
	}

	env, err := c.Do(r)
	if err != nil {
		// handle err
		panic(err)
	}

	resp := new(GetVideoSourcesResponse)
	if err := env.Body.Unmarshal(resp); err != nil {
		// handle err
		panic(err)
	}

	// check if media ver20 service is supported so newer encodings will be returned
	if s := services.URL(onvif.NamespaceMedia2); s != "" {
		mediaNS = onvif.NamespaceMedia2
		mediaURL = s
	}

	for _, source := range resp.VideoSources {
		fmt.Printf("Found video source \"%s\": (%dx%d@%0.2f)\n", source.Token, source.Width, source.Height, source.Framerate)

		r := &onvif.Request{
			URL:        mediaURL,
			Namespaces: soap.Namespaces{"trt": mediaNS},
			Body:       &GetVideoSourceModes{Token: source.Token},
		}

		env, err := c.Do(r)
		if err != nil {
			// handle err
			panic(err)
		}

		resp := new(GetVideoSourceModesResponse)
		if err := env.Body.Unmarshal(resp); err != nil {
			// handle err
			panic(err)
		}
		for _, mode := range resp.VideoSourceModes {
			fmt.Printf("\tFound mode \"%s\": (%dx%d@%0.2f); Codecs: %s\n", mode.Token, mode.MaxWidth, mode.MaxHeight, mode.MaxFramerate, strings.Join(mode.Encodings, ", "))
		}
	}

	// Output:
	// Found video source "VideoSource": (3096x2202@30.00)
	// 		Found mode "5M_1x1_FISHEYE": (2192x2192@30.00); Codecs: JPEG, H264, H265
	// 		Found mode "5M_16x9_PANORAMA": (1920x1080@30.00); Codecs: JPEG, H264, H265
	// 		Found mode "5M_16x9_WPANORAMA": (1920x1080@30.00); Codecs: JPEG, H264, H265
	// 		Found mode "5M_4x3_PTZ_QUAD_CEILING": (1600x1200@30.00); Codecs: JPEG, H264, H265
	// 		Found mode "5M_4x3_PTZ_QUAD_WALL": (1600x1200@30.00); Codecs: JPEG, H264, H265
	// 		Found mode "5M_4x3_PTZ_SINGLE_CEILING": (1600x1200@30.00); Codecs: JPEG, H264, H265
	// 		Found mode "5M_4x3_PTZ_SINGLE_WALL": (1600x1200@30.00); Codecs: JPEG, H264, H265
	// 		Found mode "5M_FISHEYE_PANORAMA": (2192x2192@30.00); Codecs: JPEG, H264, H265
	// 		Found mode "5M_FISHEYE_WPANORAMA": (2192x2192@30.00); Codecs: JPEG, H264, H265
	// 		Found mode "5M_FISHEYE_QUAD": (2192x2192@30.00); Codecs: JPEG, H264, H265
	// 		Found mode "5M_FISHEYE_SINGLE": (2192x2192@30.00); Codecs: JPEG, H264, H265
	// 		Found mode "5M_4x3_4STREAM": (1600x1200@30.00); Codecs: H264, H265
	// 		Found mode "5M_1x1_FISHEYE_25fps": (2192x2192@25.00); Codecs: JPEG, H264, H265
	// 		Found mode "5M_16x9_PANORAMA_25fps": (1920x1080@25.00); Codecs: JPEG, H264, H265
	// 		Found mode "5M_16x9_WPANORAMA_25fps": (1920x1080@25.00); Codecs: JPEG, H264, H265
	// 		Found mode "5M_4x3_PTZ_QUAD_CEILING_25fps": (1600x1200@25.00); Codecs: JPEG, H264, H265
	// 		Found mode "5M_4x3_PTZ_QUAD_WALL_25fps": (1600x1200@25.00); Codecs: JPEG, H264, H265
	// 		Found mode "5M_4x3_PTZ_SINGLE_CEILING_25fps": (1600x1200@25.00); Codecs: JPEG, H264, H265
	// 		Found mode "5M_4x3_PTZ_SINGLE_WALL_25fps": (1600x1200@25.00); Codecs: JPEG, H264, H265
	// 		Found mode "5M_FISHEYE_PANORAMA_25fps": (2192x2192@25.00); Codecs: JPEG, H264, H265
	// 		Found mode "5M_FISHEYE_WPANORAMA_25fps": (2192x2192@25.00); Codecs: JPEG, H264, H265
	// 		Found mode "5M_FISHEYE_QUAD_25fps": (2192x2192@25.00); Codecs: JPEG, H264, H265
	// 		Found mode "5M_FISHEYE_SINGLE_25fps": (2192x2192@25.00); Codecs: JPEG, H264, H265
	// 		Found mode "5M_4x3_4STREAM_25fps": (1600x1200@25.00); Codecs: H264, H265
}

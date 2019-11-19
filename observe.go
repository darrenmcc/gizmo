// Package observe provides functions
// that help with setting tracing/metrics
// in cloud providers, mainly GCP.
package gizmo

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"

	traceapi "cloud.google.com/go/trace/apiv2"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"contrib.go.opencensus.io/exporter/stackdriver/monitoredresource"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// NewStackdriverExporter will return the tracing and metrics through
// the stack driver exporter, if exists in the underlying platform.
// If exporter is registered, it returns the exporter so you can register
// it and ensure to call Flush on termination.
func NewStackdriverExporter(projectID string, onErr func(error)) (*stackdriver.Exporter, error) {
	_, svcName, svcVersion := GetServiceInfo()
	opts := getSDOpts(projectID, svcName, svcVersion, onErr)
	if opts == nil {
		return nil, nil
	}
	return stackdriver.NewExporter(*opts)
}

var projectIDOnce = sync.Once{}

// GoogleProjectID returns the GCP Project ID
// that can be used to instantiate various
// GCP clients such as Stack Driver.
func GoogleProjectID() string {
	id := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if id != "" {
		return id
	}

	projectIDOnce.Do(func() {
		url := "http://metadata.google.internal/computeMetadata/v1/project/project-id"
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			log.Printf("unable to create project id request: %s", err)
			return
		}
		req.Header.Add("Metadata-Flavor", "Google")
		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			log.Printf("unable to request project id: %s", err)
			return
		}
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("unable to read project id: %s", err)
			return
		}
		id = string(b)
		os.Setenv("GOOGLE_CLOUD_PROJECT", id)
	})

	return id
}

// IsGAE tells you whether your program is running
// within the App Engine platform.
func IsGAE() bool {
	return os.Getenv("GAE_DEPLOYMENT_ID") != ""
}

// GetGAEInfo returns the service and the version of the
// GAE application.
func GetGAEInfo() (service, version string) {
	return os.Getenv("GAE_SERVICE"), os.Getenv("GAE_VERSION")
}

// IsCloudRun tells you whether your program is running
// within the Cloud Run platform.
func IsCloudRun() bool {
	return os.Getenv("K_CONFIGURATION") != ""
}

// GetCloudRunInfo returns the service and the version of the
// Cloud Run application.
func GetCloudRunInfo() (service, version string) {
	return os.Getenv("K_SERVICE"), os.Getenv("K_REVISION")
}

// GetServiceInfo returns the GCP Project ID,
// the service name and version (GAE or through
// SERVICE_NAME/SERVICE_VERSION env vars). Note
// that SERVICE_NAME/SERVICE_VERSION are not standard but
// your application can pass them in as variables
// to be included in your trace attributes
func GetServiceInfo() (projectID, service, version string) {
	switch {
	case IsGAE():
		service, version = GetGAEInfo()
	case IsCloudRun():
		service, version = GetCloudRunInfo()
	default:
		service, version = os.Getenv("SERVICE_NAME"), os.Getenv("SERVICE_VERSION")
	}
	return GoogleProjectID(), service, version
}

// getSDOpts returns Stack Driver Options that you can pass directly
// to the OpenCensus exporter or other libraries.
func getSDOpts(projectID, service, version string, onErr func(err error)) *stackdriver.Options {
	var mr monitoredresource.Interface

	// this is so that you can export views from your local server up to SD if you wish
	creds, err := google.FindDefaultCredentials(context.Background(), traceapi.DefaultAuthScopes()...)
	if err != nil {
		return nil
	}
	canExport := IsGAE() || IsCloudRun()
	if m := monitoredresource.Autodetect(); m != nil {
		mr = m
		canExport = true
	}
	if !canExport {
		return nil
	}

	return &stackdriver.Options{
		ProjectID:         projectID,
		MonitoredResource: mr,
		MonitoringClientOptions: []option.ClientOption{
			option.WithCredentials(creds),
		},
		TraceClientOptions: []option.ClientOption{
			option.WithCredentials(creds),
		},
		OnError: onErr,
		DefaultTraceAttributes: map[string]interface{}{
			"service": service,
			"version": version,
		},
	}
}

// SkipObserve checks if the GIZMO_SKIP_OBSERVE environment variable has been populated.
// This may be used along with local development to cut down on long startup times caused
// by the 'monitoredresource.Autodetect()' call in IsGCPEnabled().
func SkipObserve() bool {
	return os.Getenv("GIZMO_SKIP_OBSERVE") != ""
}

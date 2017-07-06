package checkup

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"log"

	"github.com/PagerDuty/go-pagerduty"
)

// PagerDuty consist of all the sub components required to use PagerDuty API
type PagerDuty struct {
	Service string `json:"service_key"`
}

// NetDetails for PagerDuty Net event
type NetDetails struct {
	Endpoint  string
	Timestamp string
	Threshold string
	Max       string
	Min       string
	Median    string
	Mean      string
	All       string
	Assesment string
	Notice    string
}

// TLSDetails for PagerDuty TLS event
type TLSDetails struct {
	CommonName string
	Serial     string
	NotBefore  string
	NotAfter   string
	DNSNames   string
	Issuer     string
	ExpiresIn  string
	Assesment  string
	Notice     string
}

//Notify implements notifier interface
func (p PagerDuty) Notify(results []Result) error {
	for _, result := range results {
		if !result.Healthy {
			p.Send(result)
		}
	}
	return nil
}

//Send request via Pagerduty API to create incident
func (p PagerDuty) Send(result Result) error {
	var status string
	var details []byte
	if result.Type == "tls" {
		switch result.Status() {
		case "down":
			status = "EXPIRED"
		case "degraded":
			status = "EXPIRING"
		}
		property := result.Context.(CertProperties)
		d := TLSDetails{
			CommonName: property.CommonName,
			Serial:     fmt.Sprint(property.Serial),
			NotBefore:  fmt.Sprint(property.NotBefore),
			NotAfter:   fmt.Sprint(property.NotAfter),
			DNSNames:   fmt.Sprint(property.DNSNames),
			Issuer:     property.Issuer,
			ExpiresIn:  fmt.Sprintf("%d day(s)", -1*int(time.Since(property.NotAfter).Hours()/24)),
			Assesment:  fmt.Sprintf("%s", status),
			Notice:     result.Notice,
		}
		details, _ = json.Marshal(d)
	} else {
		status = strings.ToUpper(fmt.Sprint(result.Status()))
		stats := result.ComputeStats()
		d := NetDetails{
			Endpoint:  result.Endpoint,
			Timestamp: fmt.Sprint(time.Unix(0, result.Timestamp).UTC()),
			Threshold: fmt.Sprint(result.ThresholdRTT),
			Max:       fmt.Sprint(stats.Max),
			Min:       fmt.Sprint(stats.Min),
			Median:    fmt.Sprint(stats.Median),
			Mean:      fmt.Sprint(stats.Mean),
			All:       fmt.Sprintf("%v", result.Times),
			Assesment: fmt.Sprintf("%s", status),
			Notice:    result.Notice,
		}
		details, _ = json.Marshal(d)
	}
	var jDetails map[string]interface{}
	json.Unmarshal(details, &jDetails)
	event := pagerduty.Event{
		ServiceKey:  p.Service,
		Type:        "trigger",
		IncidentKey: result.Endpoint,
		Description: result.Title + " (" + result.Endpoint + ") is " + status,
		Client:      result.Title,
		ClientURL:   result.Endpoint,
		Details:     jDetails,
	}
	resp, err := pagerduty.CreateEvent(event)
	if err != nil {
		log.Print("ERROR: ", err)
		return err
	}
	log.Print(resp.Message, " for incident key '", resp.IncidentKey, "' with status ", resp.Status)
	return nil
}

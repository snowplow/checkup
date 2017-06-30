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

// Details for PagerDuty event
type Details struct {
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
	status := fmt.Sprint(result.Status())
	stats := result.ComputeStats()
	d := Details{
		Endpoint:  result.Endpoint,
		Timestamp: fmt.Sprint(time.Unix(0, result.Timestamp).UTC()),
		Threshold: fmt.Sprint(result.ThresholdRTT),
		Max:       fmt.Sprint(stats.Max),
		Min:       fmt.Sprint(stats.Min),
		Median:    fmt.Sprint(stats.Median),
		Mean:      fmt.Sprint(stats.Mean),
		All:       fmt.Sprintf("%v", result.Times),
		Assesment: fmt.Sprintf("%v", strings.ToUpper(status)),
		Notice:    result.Notice,
	}
	details, _ := json.Marshal(d)
	var jDetails map[string]interface{}
	json.Unmarshal(details, &jDetails)
	event := pagerduty.Event{
		ServiceKey:  p.Service,
		Type:        "trigger",
		IncidentKey: result.Endpoint,
		Description: result.Title + " (" + result.Endpoint + ") is " + d.Assesment,
		Client:      result.Title,
		ClientURL:   result.Endpoint,
		Details:     jDetails,
	}
	resp, err := pagerduty.CreateEvent(event)
	if err != nil {
		log.Print("ERROR: ", err)
		fmt.Println(resp)
		return err
	}
	log.Print(resp.Message, " for incident key '", resp.IncidentKey, "' with status ", resp.Status)
	return nil
}

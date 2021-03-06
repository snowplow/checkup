package checkup

import (
	"fmt"
	"strings"
	"time"

	"log"

	"github.com/opsgenie/opsgenie-go-sdk/alertsv2"
	"github.com/opsgenie/opsgenie-go-sdk/client"
)

// OpsGenie consist of all the sub components required to use OpsGenie API
type OpsGenie struct {
	Service string `json:"service_key"`
}

// Notify implements notifier interface
func (o OpsGenie) Notify(results []Result) error {
	for _, result := range results {
		if !result.Healthy {
			o.Send(result)
		}
	}
	return nil
}

// Send request via OpsGenie API to create incident
func (o OpsGenie) Send(result Result) error {
	var status string
	details := make(map[string]string)
	if result.Type == "tls" {
		switch result.Status() {
		case "down":
			status = "EXPIRED"
		case "degraded":
			status = "EXPIRING"
		}
		property := result.Context.(CertProperties)
		details["Common name"] = property.CommonName
		details["Serial"] = fmt.Sprint(property.Serial)
		details["Valid from"] = fmt.Sprint(property.NotBefore)
		details["Valid to"] = fmt.Sprint(property.NotAfter)
		details["DNS names"] = fmt.Sprint(property.DNSNames)
		details["Issuer"] = property.Issuer
		details["Expires in"] = fmt.Sprintf("%d day(s)", -1*int(time.Since(property.NotAfter).Hours()/24))
		details["Timestamp"] = fmt.Sprint(time.Unix(0, result.Timestamp).UTC())
	} else {
		status = strings.ToUpper(fmt.Sprint(result.Status()))
		stats := result.ComputeStats()
		details["Endpoint"] = result.Endpoint
		details["Timestamp"] = fmt.Sprint(time.Unix(0, result.Timestamp).UTC())
		details["Threshold"] = fmt.Sprint(result.ThresholdRTT)
		details["Max"] = fmt.Sprint(stats.Max)
		details["Min"] = fmt.Sprint(stats.Min)
		details["Median"] = fmt.Sprint(stats.Median)
		details["Mean"] = fmt.Sprint(stats.Mean)
		details["All"] = fmt.Sprintf("%v", result.Times)
		details["Assessment"] = fmt.Sprint(status)
	}
	if result.Notice != "" {
		details["Notice"] = result.Notice
	}
	// Some OpsGenie accounts don't support tags
	// Adding tags to details
	for k, v := range result.Tags {
		details[k] = v
	}

	opsgenie := new(client.OpsGenieClient)
	opsgenie.SetAPIKey(o.Service)
	alert, _ := opsgenie.AlertV2()

	// For OpsGenie accounts that support tags
	tags := make([]string, len(result.Tags))
	for _, v := range result.Tags {
		tags = append(tags, v)
	}

	request := alertsv2.CreateAlertRequest{
		Message:     result.Title + " (" + result.Endpoint + ") is " + status,
		Alias:       result.Endpoint,
		Description: "Alert generated by Checkup",
		Tags:        tags,
		Details:     details,
		Entity:      result.Type,
		Source:      "Checkup",
		User:        "Checkup",
	}

	response, err := alert.Create(request)
	if err != nil {
		log.Print("ERROR: ", err)
		return err
	}
	log.Printf("Create request (%s) for %s", response.RequestID, result.Endpoint)

	return nil
}

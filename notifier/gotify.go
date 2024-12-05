package notifier

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/0x2142/frigate-notify/config"
	"github.com/0x2142/frigate-notify/models"
	"github.com/0x2142/frigate-notify/util"
)

// gotifyError defines structure of Gotify error messages
type gotifyError struct {
	Error            string `json:"error"`
	ErrorCode        int    `json:"errorCode"`
	ErrorDescription string `json:"errorDescription"`
}

// gotifyPayload defines structure of Gotify push messages
type gotifyPayload struct {
	Message  string `json:"message"`
	Title    string `json:"title,omitempty"`
	Priority int    `json:"priority,omitempty"`
	Extras   struct {
		ClientDisplay struct {
			ContentType string `json:"contentType,omitempty"`
		} `json:"client::display"`
		ClientNotification struct {
			BigImageURL string `json:"bigImageUrl,omitempty"`
		} `json:"client::notification"`
	} `json:"extras,omitempty"`
}

// SendGotifyPush forwards alert messages to Gotify push notification server
func SendGotifyPush(event models.Event) {
	var snapshotURL string
	if config.ConfigData.Frigate.PublicURL != "" {
		snapshotURL = config.ConfigData.Frigate.PublicURL + "/api/events/" + event.ID + "/snapshot.jpg"
	} else {
		snapshotURL = config.ConfigData.Frigate.Server + "/api/events/" + event.ID + "/snapshot.jpg"
	}
	// Build notification
	var message string
	if config.ConfigData.Alerts.Gotify.Template != "" {
		message = renderMessage(config.ConfigData.Alerts.Gotify.Template, event, "message", "Gotify")
	} else {
		message = renderMessage("markdown", event, "message", "Gotify")
	}

	if event.HasSnapshot {
		message += fmt.Sprintf("\n\n![](%s)", snapshotURL)
	}
	title := renderMessage(config.ConfigData.Alerts.General.Title, event, "title", "Gotify")
	payload := gotifyPayload{
		Message:  message,
		Title:    title,
		Priority: 5,
	}
	payload.Extras.ClientDisplay.ContentType = "text/markdown"
	payload.Extras.ClientNotification.BigImageURL = snapshotURL

	data, err := json.Marshal(payload)
	if err != nil {
		log.Warn().
			Str("event_id", event.ID).
			Str("provider", "Gotify").
			Err(err).
			Msg("Unable to send alert")
		config.Internal.Status.Notifications.Gotify[0].NotifFailure(err.Error())
		return
	}

	gotifyURL := fmt.Sprintf("%s/message?token=%s&", config.ConfigData.Alerts.Gotify.Server, config.ConfigData.Alerts.Gotify.Token)

	header := map[string]string{"Content-Type": "application/json"}
	response, err := util.HTTPPost(gotifyURL, config.ConfigData.Alerts.Gotify.Insecure, data, "", header)
	if err != nil {
		log.Warn().
			Str("event_id", event.ID).
			Str("provider", "Gotify").
			Err(err).
			Msg("Unable to send alert")
		config.Internal.Status.Notifications.Gotify[0].NotifFailure(err.Error())
		return
	}
	// Check for errors:
	if strings.Contains(string(response), "error") {
		var errorMessage gotifyError
		json.Unmarshal(response, &errorMessage)
		log.Warn().
			Str("event_id", event.ID).
			Str("provider", "Gotify").
			Msgf("Unable to send alert: %v - %v", errorMessage.Error, errorMessage.ErrorDescription)
		config.Internal.Status.Notifications.Gotify[0].NotifFailure(errorMessage.ErrorDescription)
		return
	}
	log.Info().
		Str("event_id", event.ID).
		Str("provider", "Gotify").
		Msg("Alert sent")
	config.Internal.Status.Notifications.Gotify[0].NotifSuccess()
}

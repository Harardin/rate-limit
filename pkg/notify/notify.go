package notify

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/Harardin/rate-limit/pkg/log"
)

type Service struct {
	emailURL string
	url      string
	token    string
	cli      *http.Client
	l        log.Logger
}

// NewNotifyService creates new notification service
func NewNotifyService(notifyURL, emailURL, notifyToken string, l log.Logger) *Service {
	return &Service{
		emailURL: emailURL,
		url:      notifyURL,
		token:    notifyToken,
		cli:      http.DefaultClient,
		l:        l,
	}
}

type notifyMessage struct {
	Receivers map[string][]string `json:"receivers"`
	Text      string              `json:"text"`
}

// SendNotification sends notification to receivers via ids in their nets
// Example:
// ["slack"]["1", "2", "3"]
// ["telegram"]["1", "2", "3"]
func (s *Service) SendNotification(receivers map[string][]string, message string) ([]byte, error) {
	n := notifyMessage{
		Receivers: receivers,
		Text:      message,
	}
	msg, err := json.Marshal(n)
	if err != nil {
		s.l.Error("failed to marshal json in send notification function", err)
		return nil, err
	}

	req, err := http.NewRequest("POST", s.url, bytes.NewBuffer(msg))
	if err != nil {
		s.l.Error("failed to create new request", err)
		return nil, err
	}

	req.Header.Set("Content-type", "application/json")
	req.Header.Set("X-API-TOKEN", s.token)

	resp, err := s.cli.Do(req)
	if err != nil {
		s.l.Error("failed to make request to notification service", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.l.Error("failed to read notify service response body", err)
		return nil, err
	}

	return body, nil
}

type emailNotification struct {
	Receivers map[string][]string `json:"receivers"`
	Subject   string              `json:"subject"`
	Type      string              `json:"type"` // Template name of the message
	Data      string              `json:"data"` // Info to place inside template to use
}

// SendEmailNotification
// Sends email notifications similar to send notifications
// "subject" is subject of the message
// "template" is template to use
// "data" data to place inside html template of the message
func (s *Service) SendEmailNotification(receivers map[string][]string, subject, template, data string) ([]byte, error) {
	n := emailNotification{
		Receivers: receivers,
		Subject:   subject,
		Type:      template,
		Data:      data,
	}

	msg, err := json.Marshal(n)
	if err != nil {
		s.l.Error("failed to marshal json in send email function", err)
		return nil, err
	}

	req, err := http.NewRequest("POST", s.emailURL, bytes.NewBuffer(msg))
	if err != nil {
		s.l.Error("failed to create new request", err)
		return nil, err
	}

	req.Header.Set("Content-type", "application/json")
	req.Header.Set("X-API-TOKEN", s.token)

	resp, err := s.cli.Do(req)
	if err != nil {
		s.l.Error("failed to make request to email notification service", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.l.Error("failed to read notify service response body", err)
		return nil, err
	}

	return body, nil
}

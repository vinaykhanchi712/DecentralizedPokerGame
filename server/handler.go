package server

import (
	"io"

	"github.com/sirupsen/logrus"
)

type Handler interface {
	HandleMessage(*Message) error
}

type DefaultHandler struct {
}

func (h *DefaultHandler) HandleMessage(m *Message) error {
	b, err := io.ReadAll(m.Payload)
	if err != nil {
		return err
	}
	logrus.Info("Message from ", m.From, ":", string(b))
	return nil
}

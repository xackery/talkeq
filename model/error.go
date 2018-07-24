package model

import "fmt"

// ErrAuth is an authentication related error
type ErrAuth struct {
	Message string
}

// Error satisfies the error type interface
func (e ErrAuth) Error() string {
	if e.Message == "" {
		e.Message = "authentication failed"
	}
	return e.Message
}

// ErrMessage is a failure to send a message
type ErrMessage struct {
	Message *ChannelMessage
}

// Error satisfies the error type interface
func (e ErrMessage) Error() string {
	return fmt.Sprintf("Failed to send message: %s", e.Message.Message)
}

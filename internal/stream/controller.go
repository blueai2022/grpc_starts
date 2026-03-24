package stream

import "context"

type Controller interface {
	StartSession(ctx context.Context)
}

type controller struct {
}

// TODO: consider add ControllerConfig if the expanded example needs it
func NewController() (Controller, error) {
	return &controller{}, nil
}

// TODO: return Session
func (c *controller) StartSession(ctx context.Context) {
	// TODO:
}

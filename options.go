package sensonet

import "net/http"

type ConnOption func(*Connection)

func WithHttpClient(client *http.Client) ConnOption {
	return func(c *Connection) {
		c.client = client
	}
}

type CtrlOption func(*Controller)

func WithLogger(logger Logger) CtrlOption {
	return func(c *Controller) {
		c.logger = logger
	}
}

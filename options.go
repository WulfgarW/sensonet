package sensonet

import "net/http"

type Option func(*Connection)

func WithLogger(logger Logger) Option {
	return func(c *Connection) {
		c.logger = logger
	}
}

func WithHttpClient(client *http.Client) Option {
	return func(c *Connection) {
		c.client = client
	}
}

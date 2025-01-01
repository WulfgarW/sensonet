package sensonet

import "net/http"

type Option func(*Connection)

func WithHttpClient(client *http.Client) Option {
	return func(c *Connection) {
		c.client = client
	}
}

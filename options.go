package sensonet

import "net/http"

type Option func(*Connection)

func WithLogger(logger Logger) Option {
	return func(c *Connection) {
		c.logger = logger
		c.debug("sensonet.NewConnection() called with option WithLogger()")
	}
}

func WithHttpClient(client *http.Client) Option {
	return func(c *Connection) {
		c.client = client
		c.debug("sensonet.NewConnection() called with option WithHttpClient()")
	}
}

func DisableCache() Option {
	return func(c *Connection) {
		c.cacheDisabled = true
		c.debug("sensonet.NewConnection() called with option DisableCache()")
	}
}

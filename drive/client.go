package drive

import (
	"net/http"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	rc *resty.Client
}

func New(uid, cid, seid, kid string, opts ...Option) (*Client, error) {
	c := &Client{
		rc: resty.New(),
	}

	for _, o := range defaultOpts {
		o(c)
	}

	for _, o := range opts {
		o(c)
	}

	c.SetCookies(&http.Cookie{
		Name:     CookieUid,
		Value:    uid,
		Domain:   CookieDomain,
		Path:     "/",
		HttpOnly: true,
	})

	c.SetCookies(&http.Cookie{
		Name:     CookieCid,
		Value:    cid,
		Domain:   CookieDomain,
		Path:     "/",
		HttpOnly: true,
	})

	c.SetCookies(&http.Cookie{
		Name:     CookieSeid,
		Value:    seid,
		Domain:   CookieDomain,
		Path:     "/",
		HttpOnly: true,
	})

	c.SetCookies(&http.Cookie{
		Name:     CookieKid,
		Value:    kid,
		Domain:   CookieDomain,
		Path:     "/",
		HttpOnly: true,
	})

	if err := c.LoginCheck(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Client) HttpClient() *http.Client {
	return c.rc.GetClient()
}

func (c *Client) SetUserAgent(userAgent string) *Client {
	c.rc.SetHeader(UAKey, userAgent)
	return c
}

func (c *Client) SetCookies(cs ...*http.Cookie) *Client {
	c.rc.SetCookies(cs)
	return c
}

func (c *Client) NewRequest() *resty.Request {
	return c.rc.R()
}

type Option func(c *Client)

func UA(ua string) Option {
	return func(c *Client) {
		c.SetUserAgent(ua)
	}
}

var defaultOpts = []Option{
	UA(UA115Browser),
}

package twitch2go

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"

	"golang.org/x/net/context/ctxhttp"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
)

type Client struct {
	ClientID   string
	HTTPClient *http.Client
	apiURL     *url.URL
}

type doOptions struct {
	data      interface{}
	forceJSON bool
	headers   map[string]string
	context   context.Context
}

var (
	apiURL  = "https://api.twitch.tv"
	apiPath = "kraken"
)

// Error represents failures in the API. It represents a failure from the API.
type Error struct {
	Status  int
	Message string
}

func newError(resp *http.Response) *Error {
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &Error{Status: resp.StatusCode, Message: fmt.Sprintf("cannot read body, err: %v", err)}
	}
	return &Error{Status: resp.StatusCode, Message: string(data)}
}

func (e *Error) Error() string {
	return fmt.Sprintf("API error (%d): %s", e.Status, e.Message)
}

func NewClient(ClientID string) *Client {
	url, err := url.Parse(apiURL)
	if err != nil {
		log.Fatal(err)
	}
	return &Client{
		ClientID:   ClientID,
		apiURL:     url,
		HTTPClient: cleanhttp.DefaultClient(),
	}
}

// if error in context, return that instead of generic http error
func chooseError(ctx context.Context, err error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return err
	}
}

func (c *Client) do(method, urlPath string, doOptions doOptions) (*http.Response, error) {
	var params io.Reader
	if doOptions.data != nil || doOptions.forceJSON {
		buf, err := json.Marshal(doOptions.data)
		if err != nil {
			return nil, err
		}
		params = bytes.NewBuffer(buf)
	}
	httpClient := c.HTTPClient
	p := path.Join(apiPath, urlPath)
	url, err := c.apiURL.Parse(p)
	u := url.String()
	req, err := http.NewRequest(method, u, params)
	if err != nil {
		return nil, err
	}
	req.Header.Set("accept", "application/vnd.twitchtv.v3+json")
	req.Header.Set("client-id", c.ClientID)
	for k, v := range doOptions.headers {
		req.Header.Set(k, v)
	}
	ctx := doOptions.context
	if ctx == nil {
		ctx = context.Background()
	}
	resp, err := ctxhttp.Do(ctx, httpClient, req)
	if err != nil {
		return nil, chooseError(ctx, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, newError(resp)
	}
	return resp, nil
}

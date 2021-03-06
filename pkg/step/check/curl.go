package check

import (
	"fmt"
	"github.com/avast/retry-go"
	errors "github.com/rotisserie/eris"
	"github.com/solo-io/valet/pkg/api"
	"github.com/solo-io/valet/pkg/cmd"
	"github.com/solo-io/valet/pkg/render"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultCurlDelay       = "1s"
	DefaultCurlAttempts    = 10
	DefaultMethod          = "GET"
	DefaultPortForwardPort = 8080
)

var (
	UnexpectedStatusCodeError = func(statusCode int) error {
		return errors.Errorf("Curl got unexpected status code %d", statusCode)
	}
	UnexpectedResponseBodyError = func(responseBody string) error {
		return errors.Errorf("Curl got unexpected response body:\n%s", responseBody)
	}
)

// Use Curl to simulate testing an endpoint with an HTTP request using curl.
//
// If service is provided, then the URL to send the request to will be determined
// by getting the address to a service exposed in the current Kube context.
//
// If portForward is provided, then the curl will be wrapped in a port-forward, exposing
// some deployment and port to localhost. The request will be send to a localhost address.
//
// Only one of service or portForward should be provided.
//
// The request can be customized with the path, host, headers, and requestBody fields.
//
// The response can be validated with the statusCode, responseBody, and responseBodySubstring fields.
//
// Curl will by default try 10 times if the validation criteria isn't met for any reason, with a delay
// of 1 second between attempt. Customize these with the attempts and delay fields.
type Curl struct {
	Path                  string            `json:"path,omitempty"`
	Host                  string            `json:"host,omitempty"`
	Headers               map[string]string `json:"headers,omitempty"`
	StatusCode            int               `json:"statusCode,omitempty" valet:"default=200"`
	Method                string            `json:"method,omitempty" valet:"default=GET"`
	RequestBody           string            `json:"body,omitempty"`
	ResponseBody          string            `json:"responseBody,omitempty"`
	ResponseBodySubstring string            `json:"responseBodySubstring,omitempty"`
	Service               *ServiceRef       `json:"service,omitempty"`
	PortForward           *PortForward      `json:"portForward,omitempty"`
	Attempts              int               `json:"attempts,omitempty" valet:"default=10"`
	Delay                 string            `json:"delay,omitempty" valet:"default=1s"`
}

func (c *Curl) Run(ctx *api.WorkflowContext, values render.Values) error {
	if err := values.RenderFields(c, ctx.Runner); err != nil {
		return err
	}
	return c.doCurl(ctx, values)
}

func (c *Curl) GetDescription(ctx *api.WorkflowContext, values render.Values) (string, error) {
	if err := values.RenderFields(c, ctx.Runner); err != nil {
		return "", err
	}
	url, err := c.GetUrl(ctx, values)
	if err != nil {
		return "", err
	}
	str := fmt.Sprintf("Issuing http request\n%s %s", c.Method, url)
	if len(c.Headers) > 0 {
		str += fmt.Sprintf("\nHeaders: %v", c.Headers)
	}
	if c.RequestBody != "" {
		str += fmt.Sprintf("\nBody: %s", c.RequestBody)
	}
	str += fmt.Sprintf("\nExpected status: %d", c.StatusCode)
	if c.ResponseBody != "" {
		str += fmt.Sprintf("\nExpected response: %s", c.ResponseBody)
	} else if c.ResponseBodySubstring != "" {
		str += fmt.Sprintf("\nExpected response substring: %s", c.ResponseBodySubstring)
	}
	return str, nil
}

func (c *Curl) GetDocs(ctx *api.WorkflowContext, values render.Values, flags render.Flags) (string, error) {
	panic("implement me")
}

func (c *Curl) doCurl(ctx *api.WorkflowContext, values render.Values) error {
	delay, err := time.ParseDuration(c.Delay)
	if err != nil {
		return err
	}
	fullUrl, err := c.GetUrl(ctx, values)
	if err != nil {
		return err
	}

	var portForwardCmd *cmd.CommandStreamHandler
	if c.PortForward != nil {
		handler, err := c.PortForward.Initiate(ctx, values)
		if err != nil {
			return err
		}
		portForwardCmd = handler

		go func() {
			_ = handler.StreamHelper(nil)
		}()

		cmd.Stdout().Println("Initiated port forward")
	}

	curlErr := retry.Do(func() error {
		req, err := c.GetHttpRequest(fullUrl)
		if err != nil {
			return err
		}
		responseBody, statusCode, err := ctx.Runner.Request(req)
		if err != nil {
			return err
		}
		if c.StatusCode != statusCode {
			return UnexpectedStatusCodeError(statusCode)
		}
		if c.ResponseBody != "" && strings.TrimSpace(responseBody) != strings.TrimSpace(c.ResponseBody) {
			return UnexpectedResponseBodyError(responseBody)
		}

		if c.ResponseBodySubstring != "" && !strings.Contains(strings.TrimSpace(responseBody), strings.TrimSpace(c.ResponseBodySubstring)) {
			return UnexpectedResponseBodyError(responseBody)
		}

		cmd.Stdout().Println("Curl successful")
		return nil
	}, retry.Delay(delay), retry.Attempts(uint(c.Attempts)), retry.DelayType(retry.FixedDelay), retry.LastErrorOnly(true))

	if portForwardCmd != nil {
		_ = ctx.Runner.Kill(portForwardCmd.Process.Process)
	}
	return curlErr
}

func (c *Curl) GetUrl(ctx *api.WorkflowContext, values render.Values) (string, error) {
	if c.Service != nil {
		ipAndPort, err := c.Service.GetAddress(ctx, values)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s://%s%s", c.Service.Port, ipAndPort, c.Path), nil
	} else if c.PortForward != nil {
		return fmt.Sprintf("http://localhost:%d%s", c.PortForward.Port, c.Path), nil
	}
	return "", errors.Errorf("Must specify either service or portForward")
}

func (c *Curl) GetHttpRequest(url string) (*http.Request, error) {
	var body io.Reader
	if c.RequestBody != "" {
		body = strings.NewReader(c.RequestBody)
	}
	req, err := http.NewRequest(c.Method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header = make(http.Header)
	for k, v := range c.Headers {
		req.Header[k] = []string{v}
	}
	if c.Host != "" {
		req.Host = c.Host
	}
	return req, nil
}

type PortForward struct {
	Namespace      string `json:"namespace,omitempty" valet:"key=Namespace"`
	DeploymentName string `json:"deploymentName,omitempty"`
	Port           int    `json:"port,omitempty" valet:"default=8080"`
}

func (p *PortForward) Initiate(ctx *api.WorkflowContext, values render.Values) (*cmd.CommandStreamHandler, error) {
	port := fmt.Sprintf("%d", p.Port)
	deployment := fmt.Sprintf("deploy/%s", p.DeploymentName)
	kubectl := cmd.New().Kubectl().With("port-forward").Namespace(p.Namespace).With(deployment, port).Cmd()
	return ctx.Runner.Stream(kubectl)
}

type ServiceRef struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty" valet:"key=Namespace"`
	Port      string `json:"port,omitempty" valet:"default=http"`
}

func (s *ServiceRef) GetAddress(ctx *api.WorkflowContext, values render.Values) (string, error) {
	if err := values.RenderFields(s, ctx.Runner); err != nil {
		return "", err
	}
	return ctx.KubeClient.GetIngressAddress(s.Name, s.Namespace, s.Port)
}

func (s *ServiceRef) GetIp(ctx *api.WorkflowContext, values render.Values) (string, error) {
	url, err := s.GetAddress(ctx, values)
	if err != nil {
		return "", err
	}
	parts := strings.Split(url, ":")
	if len(parts) <= 2 {
		return parts[0], nil
	}
	return "", errors.Errorf("Unexpected url %s", url)
}

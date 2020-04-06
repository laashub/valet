package gloo_rate_limiting_test

import (
	"context"
	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/solo-io/go-utils/testutils"
	"github.com/solo-io/valet/pkg/step/check"
	"github.com/solo-io/valet/pkg/step/helm"
	"github.com/solo-io/valet/pkg/step/kubectl"
	"github.com/solo-io/valet/pkg/workflow"
	"io/ioutil"
	"os"
	"testing"
)

func TestRateLimit(t *testing.T) {
	RegisterFailHandler(Fail)
	testutils.RegisterPreFailHandler(
		func() {
			testutils.PrintTrimmedStack()
		})
	testutils.RegisterCommonFailHandlers()
	RunSpecs(t, "Rate Limit Suite")
}

var _ = Describe("Rate limit", func() {

	const (
		// Messenger, 311
		token1 = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJNZXNzZW5nZXIiLCJudW1iZXIiOiIzMTEifQ.svbQgUcAUuKHlf6U8in0O3DPGuAIQqgsPv83UIoof1ZnTjOdidqhC-i1p94bLzt67NW5NU_GICZNJU21ZRL3Dmb2ZU8Ee6t708S9rBq3z6hvHt_H-2LuYOfEmj44GqHmwAQm47p4xCaL-3DCZuoFpGUJkB6YCEf5p-r-iWYe76W7WXLqA9LJwmcnZDgasLGlFuf0sTjDzD2-dilFQhY-QFLhQ7iHjmSA6-DHqd021EhsiSrs-pb9Br9e7t39QmUqZM13SMi0VA19oyK6ORNF8zndntPf2KJ2y5M7Pf8tUi2eKTkTA_CpTjFrbsY5KsehA4V1lt-Z4QDukiVtXgSMmg"
		// Whatsapp, 311
		token2 = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJXaGF0c2FwcCIsIm51bWJlciI6IjMxMSJ9.HpZKZZ6NG9Zy8R7FD87G7A6bY03xiHSub1NqADC7uCGJZM5k6Rvk4_AcKsHYrGIlSIONoPxv63gvEuesPtqP1KseBrjuNDYJ9hmgAS6E-s8IGcxhL4h5Urm_GWBlAOZbnYRBv26spEqbkpPMttmbne4mq8K8najlMMO2WbLXO0G3XSau--HTyy28rBCNrww1Nz-94Rv4brnka4rGgTb8262Qz-CJZDqhenzT9OSIkUcDTA9EkC1b3sJ_fMB1w06yzW2Ey5SCAaByf6ARtJfApmZwC6dOOlgvBw7NJQFnXOHl22r-_1gRanT2xOzWsAHjSdQjNW1ohIjyiDqrlnCKEg"
		// Whatsapp, 411
		token3 = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzb2xvLmlvIiwic3ViIjoiMTIzNDU2Nzg5MCIsInR5cGUiOiJXaGF0c2FwcCIsIm51bWJlciI6IjQxMSJ9.nKxJufSAaW7FcM5qhUVXicn55n5tUCwVHElsnE_EfTYjveAbt7VytcrnihFZctUacrK4XguXb3HPbkb4rQ5wuS2BXoJLNJSao_9N9XtTMabGnpBp9M88dUQ7D-H2nAp-ufcbcQntl5B-gVzTcKwuWckiiMS60gdDMJ2MVcqXskeuftGGt8-Qyygi5NV5eHrlVx6I3McsBkwaw1mxgBEDhMPkgM3PTAcwfihJMdO9T25wY4APwuGB2bTyZyJ86L6xRvu-yMVHS5HouEQY--Xp-AMCbJW1Da-tyCJRBUqw8HIGEOp9wIjPNcPvZ5AZkQ1kvseSVBvtRX-QJXlHBHU6Og"
	)

	ctx := workflow.DefaultContext(context.TODO())

	installGloo := func() *workflow.Step {
		return &workflow.Step{
			InstallHelmChart: &helm.InstallHelmChart{
				ReleaseName: "gloo",
				ReleaseUri:  "https://storage.googleapis.com/gloo-ee-helm/charts/gloo-ee-1.3.0.tgz",
				Namespace:   "gloo-system",
				WaitForPods: true,
				Set: map[string]string{
					"license_key": "env:LICENSE_KEY",
				},
			},
		}
	}

	gatewayProxy := func() *check.ServiceRef {
		return &check.ServiceRef{
			Namespace: "gloo-system",
			Name:      "gateway-proxy",
		}
	}

	initialCurl := func() *workflow.Step {
		return &workflow.Step{
			Curl: &check.Curl{
				Service:      gatewayProxy(),
				Path:         "/sample-route-1",
				StatusCode:   200,
				ResponseBody: `[{"id":1,"name":"Dog","status":"available"},{"id":2,"name":"Cat","status":"pending"}]`,
			},
		}
	}

	patchSettings := func(path string) *workflow.Step {
		return &workflow.Step{
			Patch: &kubectl.Patch{
				Name:      "default",
				Namespace: "gloo-system",
				KubeType:  "settings",
				PatchType: "merge",
				Path:      path,
			},
		}
	}

	rateLimitedCurl := func() *workflow.Step {
		return &workflow.Step{
			Curl: &check.Curl{
				Service:    gatewayProxy(),
				Path:       "/sample-route-1",
				StatusCode: 429,
			},
		}
	}

	curlWithHeaders := func(status int, typeHeader, numberHeader string) *workflow.Step {
		return &workflow.Step{
			Curl: &check.Curl{
				Service:    gatewayProxy(),
				Path:       "/sample-route-1",
				StatusCode: status,
				Headers: map[string]string{
					"x-type":   typeHeader,
					"x-number": numberHeader,
				},
			},
		}
	}

	curlWithToken := func(status int, token string) *workflow.Step {
		return &workflow.Step{
			Curl: &check.Curl{
				Service:    gatewayProxy(),
				Path:       "/sample-route-1",
				StatusCode: status,
				Headers: map[string]string{
					"x-token": token,
				},
			},
		}
	}

	curlForEventualRateLimit := func(status int, token string) *workflow.Step {
		step := curlWithToken(status, token)
		step.Curl.Attempts = 100
		step.Curl.Delay = "100ms"
		return step
	}

	getWorkflow := func() *workflow.Workflow {
		return &workflow.Workflow{
			Steps: []*workflow.Step{
				installGloo(),
				// Part 1: Deploy the app
				workflow.Apply("petstore.yaml"),
				workflow.WaitForPods("default"),
				workflow.Apply("vs-petstore-1.yaml"),
				initialCurl(),
				// Part 2: Set up initial RL
				patchSettings("settings-patch-1.yaml"),
				workflow.Apply("vs-petstore-2.yaml"),
				rateLimitedCurl(),
				// Part 3: Set up complex rules with priority
				patchSettings("settings-patch-2.yaml"),
				workflow.Apply("vs-petstore-3.yaml"),
				curlWithHeaders(429, "Messenger", "311"),
				curlWithHeaders(429, "Whatsapp", "311"),
				curlWithHeaders(200, "Whatsapp", "411"),
				// Part 4: Add JWT filter to set headers from JWT claims
				workflow.Apply("vs-petstore-4.yaml"),
				curlWithToken(429, token1),
				curlWithToken(429, token2),
				curlWithToken(200, token3),
				curlForEventualRateLimit(429, token3),
			},
		}
	}

	It("runs", func() {
		globalConfig, err := workflow.LoadDefaultGlobalConfig(ctx.FileStore)
		Expect(err).To(BeNil())
		err = workflow.LoadEnv(globalConfig)
		Expect(err).To(BeNil())
		err = getWorkflow().Run(ctx)
		Expect(err).To(BeNil())
	})

	It("can serialize as and deserialize from yaml", func() {
		initial := getWorkflow()
		bytes, err := yaml.Marshal(initial)
		Expect(err).To(BeNil())
		err = ioutil.WriteFile("workflow.yaml", bytes, os.ModePerm)
		Expect(err).To(BeNil())
		deserialized := &workflow.Workflow{}
		err = yaml.UnmarshalStrict(bytes, deserialized, yaml.DisallowUnknownFields)
		Expect(err).To(BeNil())
		Expect(deserialized).To(Equal(initial))
	})
})
package resource

import (
	"context"
	"strings"
	"text/template"

	"github.com/solo-io/valet/cli/internal/ensure/cmd"
)

type Template struct {
	Path   string `yaml:"path"`
	Values Values `yaml:"values"`
}

func (t *Template) Ensure(ctx context.Context, input InputParams, command cmd.Factory) error {
	input = input.MergeValues(t.Values)
	cmd.Stdout().Println("Ensuring template %s %s", t.Path, input.Values.ToString())
	rendered, err := t.Load(input)
	if err != nil {
		return err
	}
	return command.Kubectl().ApplyStdIn(rendered).Cmd().Run(ctx)
}

func (t *Template) Teardown(ctx context.Context, input InputParams, command cmd.Factory) error {
	input = input.MergeValues(t.Values)
	cmd.Stdout().Println("Tearing down template %s %s", t.Path, input.Values.ToString())
	rendered, err := t.Load(input)
	if err != nil {
		return err
	}
	return command.Kubectl().DeleteStdIn(rendered).Cmd().Run(ctx)
}

func (t *Template) Load(input InputParams) (string, error) {
	tmpl, err := LoadFile(t.Path)
	if err != nil {
		return "", err
	}
	return LoadTemplate(tmpl, input.Values)
}

func LoadTemplate(tmpl string, values Values) (string, error) {
	parsed, err := template.New("").Parse(tmpl)
	if err != nil {
		return "", err
	}
	out := strings.Builder{}
	vals, err := renderValues(values)
	if err != nil {
		return "", err
	}
	err = parsed.Execute(&out, vals)
	return out.String(), err
}

func renderValues(values Values) (map[string]interface{}, error) {
	vals := make(map[string]interface{})
	for k := range values {
		v, err := values.GetValue(k)
		if err != nil {
			return nil, err
		}
		vals[k] = v
	}
	return vals, nil
}

func LoadFile(path string) (string, error) {
	b, err := loadBytesFromPath(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

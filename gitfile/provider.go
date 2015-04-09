package gitfile

import (
	"github.com/hashicorp/errwrap"
	"os/exec"
	"fmt"
	"github.com/hashicorp/terraform/helper/hashcode"
	"io/ioutil"
	"path"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
	"github.com/hashicorp/terraform/terraform"
)

const CommitBodyHeader string = "The following files are managed by terraform:"

func Provider() terraform.ResourceProvider {
	file_resource := fileResource()
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
		},
		ResourcesMap: map[string]*schema.Resource{
			"gitfile_checkout": checkoutResource(),
			"gitfile_commit": commitResource(file_resource),
		},
		ConfigureFunc: gitfileConfigure,
	}
}

func gitfileConfigure(data *schema.ResourceData) (interface{}, error) {
	config := &gitfileConfig {
	}
	return config, nil
}

type gitfileConfig struct {
}

func fileResource() *schema.Resource {
	return &schema.Resource {
		Schema: map[string]*schema.Schema{
			"path": &schema.Schema{
				Type: schema.TypeString,
				Required: true,
			},
			"contents": &schema.Schema{
				Type: schema.TypeString,
				Required: true,
			},
		},
	}
}

func fileCreateUpdate(checkout_dir, filepath, contents string) error {
	if err := ioutil.WriteFile(path.Join(checkout_dir, filepath), []byte(contents), 0666); err != nil {
		return err
	}

	return nil
}

func fileRead(checkout_dir, filepath string) (map[string]interface{}, error) {
	if content_bytes, err := ioutil.ReadFile(path.Join(checkout_dir, filepath)); err != nil {
		return nil, err
	} else {
		return map[string]interface{}{
			"contents": string(content_bytes),
			"path": filepath,
		}, nil
	}

	return nil, nil
}

func hashFile(v interface{}) int {
	switch v := v.(type) {
	default:
		panic(fmt.Sprintf("unexpectedtype %T", v))
	case map[string]string:
		return hashcode.String(v["path"])
	case map[string]interface{}:
		return hashcode.String(v["path"].(string))
	}
	return -1
}

func gitCommand(checkout_dir string, args ...string) ([]byte, error) {
	command := exec.Command("git", args...)
	command.Dir = checkout_dir
	out, err := command.CombinedOutput()
	if err != nil {
		return out, errwrap.Wrapf(fmt.Sprintf("Error while running git %s: {{err}}\nWorking dir: %s\nOutput: %s", strings.Join(args, " "), checkout_dir, string(out)), err)
	} else {
		return out, err
	}
}

func flatten(args ...interface{}) []string {
	ret := make([]string, 0, len(args))

	for _, arg := range args {
		switch arg := arg.(type) {
		default:
			panic("can only handle strings and []strings")
		case string:
			ret = append(ret, arg)
		case []string:
			ret = append(ret, arg...)
		}
	}

	return ret
}


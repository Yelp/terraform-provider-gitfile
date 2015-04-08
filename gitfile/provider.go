package gitfile

import (
	"github.com/hashicorp/errwrap"
	"os/exec"
	"fmt"
	"github.com/hashicorp/terraform/helper/hashcode"
	"io/ioutil"
	"os"
	"path"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
	"syscall"
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

func checkoutResource() *schema.Resource {
	return &schema.Resource {
		Schema: map[string]*schema.Schema{
			"path": &schema.Schema{
				Type: schema.TypeString,
				Optional: true,
			},
			"repo": &schema.Schema{
				Type: schema.TypeString,
				Required: true,
			},
			"branch": &schema.Schema{
				Type: schema.TypeString,
				Optional: true,
				Default: "master",
			},
			"head": &schema.Schema{
				Type: schema.TypeString,
				Computed: true,
			},
		},
		Create: CheckoutCreate,
		Read: CheckoutRead,
		Update: nil,
		Delete: CheckoutDelete,
	}
}

func CheckoutCreate(d *schema.ResourceData, meta interface{}) error {
	checkout_dir := d.Get("path").(string)
	repo := d.Get("repo").(string)
	branch := d.Get("branch").(string)

	if err := os.MkdirAll(checkout_dir, 0755); err != nil {
		return err
	}

	if _, err := gitCommand(checkout_dir, "clone", "-b", branch, "--", repo, "."); err != nil {
		return err
	}
	var head string
	if out, err := gitCommand(checkout_dir, "rev-parse", "HEAD"); err != nil {
		return err
	} else {
		head = strings.TrimRight(string(out), "\n")
	}

	d.Set("head", head)
	d.SetId(checkout_dir)
	return nil
}

func CheckoutRead(d *schema.ResourceData, meta interface{}) error {
	checkout_dir := d.Id()
	var repo string
	var branch string
	var head string

	if out, err := gitCommand(checkout_dir, "config", "--get", "remote.origin.url"); err != nil {
		return err
	} else {
		repo = strings.TrimRight(string(out), "\n")
	}
	if out, err := gitCommand(checkout_dir, "rev-parse", "--abbrev-ref", "HEAD"); err != nil {
		return err
	} else {
		branch = strings.TrimRight(string(out), "\n")
	}

	if _, err := gitCommand(checkout_dir, "pull", "--ff-only", "origin"); err != nil {
		return err
	}

	if out, err := gitCommand(checkout_dir, "rev-parse", "HEAD"); err != nil {
		return err
	} else {
		head = strings.TrimRight(string(out), "\n")
	}

	d.Set("path", checkout_dir)
	d.Set("repo", repo)
	d.Set("branch", branch)
	d.Set("head", head)
	return nil
}

func CheckoutDelete(d *schema.ResourceData, meta interface{}) error {
	checkout_dir := d.Id()
	expected_repo := d.Get("repo").(string)
	expected_branch := d.Get("branch").(string)
	expected_head := d.Get("head").(string)

	// sanity check
	var repo string
	var branch string
	var head string

	if out, err := gitCommand(checkout_dir, "config", "--get", "remote.origin.url"); err != nil {
		return err
	} else {
		repo = strings.TrimRight(string(out), "\n")
	}
	if out, err := gitCommand(checkout_dir, "rev-parse", "--abbrev-ref", "HEAD"); err != nil {
		return err
	} else {
		branch = strings.TrimRight(string(out), "\n")
	}

	if _, err := gitCommand(checkout_dir, "pull", "--ff-only", "origin"); err != nil {
		return err
	}

	if out, err := gitCommand(checkout_dir, "rev-parse", "HEAD"); err != nil {
		return err
	} else {
		head = strings.TrimRight(string(out), "\n")
	}

	if expected_repo != repo {
		return fmt.Errorf("expected repo to be %s, was %s", expected_repo, repo)
	}
	if expected_branch != branch {
		return fmt.Errorf("expected branch to be %s, was %s", expected_branch, branch)
	}
	if expected_head != head {
		return fmt.Errorf("expected head to be %s, was %s", expected_head, head)
	}

	// more sanity checks
	if out, err := gitCommand(checkout_dir, "clean", "-dn"); err != nil {
		return err
	} else {
		if out != nil && string(out) != "" {
			return fmt.Errorf("Refusing to delete checkout %s with untracked files: %s", checkout_dir, string(out))
		}
	}
	if out, err := gitCommand(checkout_dir, "diff-index", "--exit-code", "HEAD"); err != nil {
		exitErr, isExitErr := err.(*exec.ExitError)
		if isExitErr {
			if exitErr.Sys().(syscall.WaitStatus).ExitStatus() != 1 {
				return err
			} else {
				return fmt.Errorf("Refusing to delete dirty checkout %s: %s", checkout_dir, string(out))
			}
		} else {
			return err
		}
	}

	// actually delete
	if err := os.RemoveAll(checkout_dir); err != nil {
		return err
	}

	return nil
}


func commitResource(file_resource *schema.Resource) *schema.Resource {
	return &schema.Resource {
		Schema: map[string]*schema.Schema {
			"commit_message": &schema.Schema{
				Type: schema.TypeString,
				Optional: true,
				Default: "Created by terraform gitfile_commit",
			},
			"checkout_dir": &schema.Schema {
				Type: schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"file": &schema.Schema {
				Type: schema.TypeSet,
				Required: true,
				Set: hashFile,
				Elem: file_resource,
			},
		},
		Create: CommitCreate,
		Read: CommitRead,
		Update: CommitCreate,
		Delete: CommitDelete,
	}
}

func CommitCreate(d *schema.ResourceData, meta interface{}) error {
	checkout_dir := d.Get("checkout_dir").(string)
	files := d.Get("file").(*schema.Set)
	commit_message := d.Get("commit_message").(string)
	filepaths := []string{}
	filepaths_to_commit := []string{}
	for _, file := range files.List() {
		filepath := file.(map[string]interface{})["path"].(string)
		filepaths = append(filepaths, filepath)

		if existing_content_bytes, err := ioutil.ReadFile(path.Join(checkout_dir, filepath)); err != nil && !os.IsNotExist(err) {
			return err;
		} else {
			contents := file.(map[string]interface{})["contents"].(string)
			// we only want to git add/git commit if we've changed this file
			// so if it existed before (err == nil) and the contents are the same
			// then don't bother.
			if !(err == nil && contents == string(existing_content_bytes)) {
				filepaths_to_commit = append(filepaths_to_commit, filepath)
			}
			if err := fileCreateUpdate(checkout_dir, filepath, contents); err != nil {
				return err
			}
		}
	}

	var sha string
	if _, err := gitCommand(checkout_dir, flatten("add", "--", filepaths_to_commit)...); err != nil {
		return err
	}

	commit_body := fmt.Sprintf("%s\n%s", CommitBodyHeader, strings.Join(filepaths, "\n"))
	if _, err := gitCommand(checkout_dir, flatten("commit", "-m", commit_message, "-m", commit_body, "--allow-empty", "--", filepaths_to_commit)...); err != nil {
		return err
	}

	if _, err := gitCommand(checkout_dir, "push", "origin", "HEAD"); err != nil {
		return err
	}

	if out, err := gitCommand(checkout_dir, "rev-parse", "HEAD"); err != nil {
		return err
	} else {
		sha = strings.TrimRight(string(out), "\n")
	}

	d.SetId(fmt.Sprintf("%s %s", sha, checkout_dir))
	return nil
}

func CommitRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func CommitDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
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


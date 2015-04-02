package gitfile

import (
	b64 "encoding/base64"
	"os/exec"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"workdir": &schema.Schema{
				Type: schema.TypeString,
				Required: true,
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"gitfile_file": fileResource(),
		},
		ConfigureFunc: gitfileConfigure,
	}
}

func gitfileConfigure(data *schema.ResourceData) (interface{}, error) {
	config := &gitfileConfig {
		workDir: data.Get("workdir").(string),
	}
	return config, nil
}

type gitfileConfig struct {
	workDir string
}

func fileResource() *schema.Resource {
	return &schema.Resource {
		Schema: map[string]*schema.Schema {
			"repo": &schema.Schema{
				Type: schema.TypeString,
				Required: true,
			},
			"path": &schema.Schema{
				Type: schema.TypeString,
				Required: true,
			},
			"contents": &schema.Schema{
				Type: schema.TypeString,
				Required: true,
			},
			"branch": &schema.Schema{
				Type: schema.TypeString,
				Optional: true,
				Default: "master",
			},
		},
		Create: FileCreate,
		Read: FileRead,
		Update: FileUpdate,
		Delete: FileDelete,
	}
}

func FileCreate(d *schema.ResourceData, meta interface{}) error {
	d.SetId(fmt.Sprintf("%s %s %s", d.Get("repo"), d.Get("branch"), d.Get("path")))
	return nil
}
func FileRead(d *schema.ResourceData, meta interface{}) error {
	splits := strings.SplitN(d.Id(), " ", 3)
	repo := splits[0]
	branch := splits[1]
	filepath := splits[2]
	workdir := meta.(*gitfileConfig).workDir

	d.Set("repo", repo)
	d.Set("branch", branch)
	d.Set("path", filepath)

	checkout_dir := path.Join(workdir, mungeGitDir(d.Id()))
	if err := shallowSparseGitCheckout(checkout_dir, repo, branch, filepath); err != nil {
		return err
	}

	contents, err := ioutil.ReadFile(path.Join(checkout_dir, filepath))
	if err != nil {
		if os.IsNotExist(err) {
			d.Set("contents", "")
		} else {
			return err
		}
	} else {
		d.Set("contents", string(contents))
	}
	return nil
}
func FileUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}
func FileDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}


func mungeGitDir(id string) string {
	return b64.URLEncoding.EncodeToString([]byte(id))
}

func shallowSparseGitCheckout(checkout_dir, repo, branch, filepath string) error {
	if err := os.MkdirAll(checkout_dir, 0755); err != nil {
		return err
	}

	// git init appears to be idempotent.
	git_init := exec.Command("git", "init")
	git_init.Dir = checkout_dir
	if err := git_init.Run(); err != nil {
		return err
	}

	git_config := exec.Command("git", "config", "core.sparsecheckout", "true")
	git_config.Dir = checkout_dir
	if err := git_config.Run(); err != nil {
		return err
	}

	scf, err := os.Create(path.Join(checkout_dir, ".git", "info", "sparse-checkout"), )
	if err != nil {
		return err
	}
	if _, err := scf.WriteString(filepath); err != nil {
		return err
	}
	if err := scf.Close(); err != nil {
		return err
	}

	git_fetch := exec.Command("git", "fetch", "--depth", "1", repo, branch)
	git_fetch.Dir = checkout_dir
	if err := git_fetch.Run(); err != nil {
		return err
	}

	git_checkout := exec.Command("git", "checkout", "--force", "FETCH_HEAD")
	git_checkout.Dir = checkout_dir
	if err := git_checkout.Run(); err != nil {
		return err
	}
	return nil
}
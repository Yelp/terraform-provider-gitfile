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
			"commit_message": &schema.Schema{
				Type: schema.TypeString,
				Optional: true,
				Default: "Created by terraform gitfile_file",
			},
		},
		Create: FileCreate,
		Read: FileRead,
		Update: FileUpdate,
		Delete: FileDelete,
	}
}

func FileCreate(d *schema.ResourceData, meta interface{}) error {
	repo := d.Get("repo").(string)
	branch := d.Get("branch").(string)
	filepath := d.Get("path").(string)
	workdir := meta.(*gitfileConfig).workDir

	d.SetId(fmt.Sprintf("%s %s %s", repo, branch, filepath))

	checkout_dir := path.Join(workdir, mungeGitDir(d.Id()))
	if err := shallowSparseGitCheckout(checkout_dir, repo, branch, filepath); err != nil {
		return err
	}

	return FileUpdate(d, meta)
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
	repo := d.Get("repo").(string)
	branch := d.Get("branch").(string)
	filepath := d.Get("path").(string)
	contents := d.Get("contents").(string)
	commit_message := d.Get("commit_message").(string)
	workdir := meta.(*gitfileConfig).workDir
	checkout_dir := path.Join(workdir, mungeGitDir(d.Id()))

	if err := ioutil.WriteFile(path.Join(checkout_dir, filepath), []byte(contents), 0666); err != nil {
		return err
	}

	// Only bother trying to commit things if the contents have changed.
	// I'm pretty sure this should be relatively accurate, since terraform will generally call FileRead before this.
	if d.HasChange("contents") {
		git_add := exec.Command("git", "add", "--intent-to-add", "--", filepath)
		git_add.Dir = checkout_dir
		if err := git_add.Run(); err != nil {
			return err
		}

		git_commit := exec.Command("git", "commit", "-m", commit_message, "--", filepath)
		git_commit.Dir = checkout_dir
		if err := git_commit.Run(); err != nil {
			return err
		}

		git_push := exec.Command("git", "push", repo, fmt.Sprintf("HEAD:%s", branch))
		git_push.Dir = checkout_dir
		if err := git_push.Run(); err != nil {
			return err
		}
	}
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

	if err := ioutil.WriteFile(
		path.Join(checkout_dir, ".git", "info", "sparse-checkout"),
		[]byte(filepath),
		0666,
	); err != nil {
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

	git_clean := exec.Command("git", "clean", "-ffdx")
	git_clean.Dir = checkout_dir
	if err := git_clean.Run(); err != nil {
		return err
	}

	return nil
}
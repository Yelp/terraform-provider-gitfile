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
	"syscall"
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
		if err := gitCommand(checkout_dir, "add", "--intent-to-add", "--", filepath); err != nil {
			return err
		}

		if err := gitCommand(checkout_dir, "commit", "-m", commit_message, "--", filepath); err != nil {
			return err
		}

		if err := gitCommand(checkout_dir, "push", repo, fmt.Sprintf("HEAD:%s", branch)); err != nil {
			return err
		}
	}
	return nil
}
func FileDelete(d *schema.ResourceData, meta interface{}) error {
	splits := strings.SplitN(d.Id(), " ", 3)
	repo := splits[0]
	branch := splits[1]
	filepath := splits[2]
	workdir := meta.(*gitfileConfig).workDir
	checkout_dir := path.Join(workdir, mungeGitDir(d.Id()))
	commit_message := d.Get("commit_message").(string)

	if err := gitCommand(checkout_dir, "rm", "--ignore-unmatch", "--", filepath); err != nil {
		return err
	}

	if err := gitCommand(checkout_dir, "diff-index", "--exit-code", "--quiet", "HEAD", "--", filepath); err != nil {
		exitErr, isExitErr := err.(*exec.ExitError)
		if isExitErr {
			if exitErr.Sys().(syscall.WaitStatus).ExitStatus() != 1 {
				return err
			} else {
				if err := gitCommand(checkout_dir, "commit", "-m", commit_message, "--", filepath); err != nil {
					return err
				}

				if err := gitCommand(checkout_dir, "push", repo, fmt.Sprintf("HEAD:%s", branch)); err != nil {
					return err
				}
			}
		} else {
			return err
		}
	}

	return nil
}

func gitCommand(checkout_dir string, args ...string) error {
	command := exec.Command("git", args...)
	command.Dir = checkout_dir
	if err := command.Run(); err != nil {
		return err
	}
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
	if err := gitCommand(checkout_dir, "init"); err != nil {
		return err
	}

	if err := gitCommand(checkout_dir, "config", "core.sparsecheckout", "true"); err != nil {
		return err
	}

	if err := ioutil.WriteFile(
		path.Join(checkout_dir, ".git", "info", "sparse-checkout"),
		[]byte(filepath),
		0666,
	); err != nil {
		return err
	}

	if err := gitCommand(checkout_dir, "fetch", "--depth", "1", repo, branch); err != nil {
		return err
	}

	if err := gitCommand(checkout_dir, "checkout", "--force", "FETCH_HEAD"); err != nil {
		return err
	}

	if err := gitCommand(checkout_dir, "clean", "-ffdx"); err != nil {
		return err
	}

	return nil
}
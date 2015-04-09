package gitfile

import (
	"os/exec"
	"fmt"
	"os"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
	"syscall"
)


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
	lockCheckout(checkout_dir)
	defer unlockCheckout(checkout_dir)

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
	lockCheckout(checkout_dir)
	defer unlockCheckout(checkout_dir)

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


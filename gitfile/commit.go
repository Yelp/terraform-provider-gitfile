package gitfile

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
)

const CommitBodyHeader string = "The following files are managed by terraform:"

func commitResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"commit_message": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Created by terraform gitfile_commit",
				ForceNew: true,
			},
			"checkout_dir": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"handle": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
		Create: CommitCreate,
		Read:   CommitRead,
		Delete: CommitDelete,
		Exists: CommitExists,
	}
}

func CommitCreate(d *schema.ResourceData, meta interface{}) error {
	checkout_dir := d.Get("checkout_dir").(string)
	lockCheckout(checkout_dir)
	defer unlockCheckout(checkout_dir)

	handle := d.Get("handle").(string)
	commit_message := d.Get("commit_message").(string)
	filepath := parseHandle(handle).path

	var sha string

	commit_body := fmt.Sprintf("%s\n%s", CommitBodyHeader, filepath)
	if _, err := gitCommand(checkout_dir, flatten("commit", "-m", commit_message, "-m", commit_body, "--allow-empty", "--", filepath)...); err != nil {
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

func CommitExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	checkoutDir := d.Get("checkout_dir").(string)
	lockCheckout(checkoutDir)
	defer unlockCheckout(checkoutDir)
	commitId := strings.Split(d.Id(), " ")[0]

	_, err := gitCommand(checkoutDir, flatten("show", commitId)...)

	if err != nil {
		return false, nil
	} else {
		return true, nil
	}

}

func CommitDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}

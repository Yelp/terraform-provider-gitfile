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
			},
			"checkout_dir": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"handles": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Set:      hashString,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
		Create: CommitCreate,
		Read:   CommitRead,
        Update: CommitCreate,
		Delete: CommitDelete,
	}
}

func CommitCreate(d *schema.ResourceData, meta interface{}) error {
	checkout_dir := d.Get("checkout_dir").(string)
	lockCheckout(checkout_dir)
	defer unlockCheckout(checkout_dir)

	handles := d.Get("handles").(*schema.Set)
	commit_message := d.Get("commit_message").(string)
	filepaths := []string{}
	for _, handle := range handles.List() {
		filepaths = append(filepaths, parseHandle(handle.(string)).path)
	}

	var sha string

	commit_body := fmt.Sprintf("%s\n%s", CommitBodyHeader, strings.Join(filepaths, "\n"))
	if _, err := gitCommand(checkout_dir, flatten("commit", "-m", commit_message, "-m", commit_body, "--allow-empty", "--", filepaths)...); err != nil {
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

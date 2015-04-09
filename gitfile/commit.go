package gitfile

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
)

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



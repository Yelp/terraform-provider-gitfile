package gitfile

import (
	b64 "encoding/base64"
	"github.com/hashicorp/errwrap"
	"os/exec"
	"fmt"
	"github.com/hashicorp/terraform/helper/hashcode"
	"io/ioutil"
	"encoding/json"
	"log"
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
			"bogus_filename": &schema.Schema{
				Type: schema.TypeString,
				Optional: true,
				Default: ".gitignore",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"gitfile_checkout": checkoutResource(),
			"gitfile_file": fileResource(),
			"gitfile_commit": commitResource(),
		},
		ConfigureFunc: gitfileConfigure,
	}
}

func gitfileConfigure(data *schema.ResourceData) (interface{}, error) {
	config := &gitfileConfig {
		workDir: data.Get("workdir").(string),
		bogusFilename: data.Get("bogus_filename").(string),
	}
	return config, nil
}

type gitfileConfig struct {
	workDir string
	bogusFilename string
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
			"checkout_dir": &schema.Schema{
				Type: schema.TypeString,
				Required: true,
			},
		},
		Create: FileCreateUpdate,
		Read: FileRead,
		Update: FileCreateUpdate,
		Delete: FileDelete,
	}
}

func FileCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	filepath := d.Get("path").(string)
	contents := d.Get("contents").(string)
	checkout_dir := d.Get("checkout_dir").(string)

	if id_bytes, err := json.Marshal([]string{checkout_dir, filepath}); err != nil {
		return err
	} else {
		d.SetId(string(id_bytes))
	}

	if err := ioutil.WriteFile(path.Join(checkout_dir, filepath), []byte(contents), 0666); err != nil {
		return err
	}

	return nil
}

func FileRead(d *schema.ResourceData, meta interface{}) error {
	list := []string{}
	if err := json.Unmarshal([]byte(d.Id()), &list); err != nil {
		return err
	}
	checkout_dir := list[0]
	filepath := list[1]

	d.Set("checkout_dir", checkout_dir)
	d.Set("path", filepath)

	if content_bytes, err := ioutil.ReadFile(path.Join(checkout_dir, filepath)); err != nil {
		return err
	} else {
		d.Set("contents", string(content_bytes))
	}

	return nil
}

func FileDelete(d *schema.ResourceData, meta interface{}) error {
	// Currently not managing deletes with this. Will make a gitfile_purgedir resource later.
	return nil
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


func commitResource() *schema.Resource {
	return &schema.Resource {
		Schema: map[string]*schema.Schema {
			"commit_message": &schema.Schema{
				Type: schema.TypeString,
				Optional: true,
				Default: "Created by terraform gitfile_commit",
			},
			"paths": &schema.Schema {
				Type: schema.TypeSet,
				Required: true,
				Set: hashString,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
		Create: CommitCreate,
		Read: CommitRead,
		Update: CommitUpdate,
		Delete: CommitDelete,
	}
}

func hashString(v interface{}) int {
	return hashcode.String(v.(string))
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

func getFilePaths(d *schema.ResourceData) []string {
	files := d.Get("file").(*schema.Set)
	filepaths := make([]string, files.Len())
	i := 0
	for _, file := range files.List() {
		filepaths[i] = file.(map[string]interface{})["path"].(string)
		i = i + 1
	}
	log.Printf("RCHOEURHOEURCHOEURCHOEURCHOEU %#v", files)
	return filepaths
}

func getFileMap(d *schema.ResourceData) map[string]map[string]interface{} {
	files := d.Get("file").(*schema.Set)
	ret := make(map[string]map[string]interface{})
	for _, fd := range files.List() {
		file := fd.(map[string]interface{})
		ret[file["path"].(string)] = file
	}
	return ret
}

func CommitCreate(d *schema.ResourceData, meta interface{}) error {
	repo := d.Get("repo").(string)
	branch := d.Get("branch").(string)
	commit_message := d.Get("commit_message").(string)
	bogus_file_name := meta.(*gitfileConfig).bogusFilename
	workdir := meta.(*gitfileConfig).workDir

	checkout_dir := path.Join(workdir, mungeGitDir(d.Id()))

	if err := shallowSparseGitCheckout(checkout_dir, repo, branch, bogus_file_name, getFilePaths(d)); err != nil {
	        return err
	}

	for filepath, filedict := range getFileMap(d) {
		if _, err := gitCommand(checkout_dir, "add", "--intent-to-add", "--", filepath); err != nil {
			return err
		}

		if err := ioutil.WriteFile(path.Join(checkout_dir, filepath), []byte(filedict["contents"].(string)), 0666); err != nil {
			return err
		}
	}

	if _, err := gitCommand(checkout_dir, flatten("commit", "-m", commit_message, "--", getFilePaths(d))...); err != nil {
		return err
	}

	if _, err := gitCommand(checkout_dir, "push", repo, fmt.Sprintf("HEAD:%s", branch)); err != nil {
		return err
	}

	fileListJson, err := json.Marshal(getFilePaths(d))
	if err != nil {
		return err
	}
	d.SetId(fmt.Sprintf("%s %s %s", repo, branch, string(fileListJson)))

	return nil
}
func CommitRead(d *schema.ResourceData, meta interface{}) error {
	splits := strings.SplitN(d.Id(), " ", 3)
	repo := splits[0]
	branch := splits[1]
	fileListJson := splits[2]

	workdir := meta.(*gitfileConfig).workDir
	bogus_file_name := meta.(*gitfileConfig).bogusFilename

	d.Set("repo", repo)
	d.Set("branch", branch)
	files := schema.NewSet(hashFile, []interface{}{})
	filelist := []string{}
	if err := json.Unmarshal([]byte(fileListJson), &filelist); err != nil {
		return err
	}

	checkout_dir := path.Join(workdir, mungeGitDir(d.Id()))
	if err := shallowSparseGitCheckout(checkout_dir, repo, branch, bogus_file_name, filelist); err != nil {
		return err
	}

	for _, filepath := range filelist {
		if filepath != "" {
			filedict := make(map[string]string)
			filedict["path"] = filepath
			contents, err := ioutil.ReadFile(path.Join(checkout_dir, filepath))
			if err != nil {
				if os.IsNotExist(err) {
					filedict["contents"] = ""
				} else {
					return err
				}
			} else {
				filedict["contents"] = string(contents)
			}
			files.Add(filedict)
		}
	}
	d.Set("file", files)
	return nil
}
func CommitUpdate(d *schema.ResourceData, meta interface{}) error {
	repo := d.Get("repo").(string)
	branch := d.Get("branch").(string)
	commit_message := d.Get("commit_message").(string)
	bogus_file_name := meta.(*gitfileConfig).bogusFilename
	workdir := meta.(*gitfileConfig).workDir

	checkout_dir := path.Join(workdir, mungeGitDir(d.Id()))

	splits := strings.SplitN(d.Id(), " ", 3)
	fileListJson := []byte(splits[2])
	filelist := []string{}
	if err := json.Unmarshal([]byte(fileListJson), &filelist); err != nil {
		return err
	}

	if err := shallowSparseGitCheckout(checkout_dir, repo, branch, bogus_file_name, flatten(getFilePaths(d), filelist)); err != nil {
	        return err
	}

	for _, filepath := range filelist {
		if _, err := gitCommand(checkout_dir, "rm", "--", filepath); err != nil {
			return err
		}
	}
	for filepath, filedict := range getFileMap(d) {
		if _, err := gitCommand(checkout_dir, "add", "--intent-to-add", "--", filepath); err != nil {
			return err
		}

		if err := ioutil.WriteFile(path.Join(checkout_dir, filepath), []byte(filedict["contents"].(string)), 0666); err != nil {
			return err
		}
	}

	if _, err := gitCommand(checkout_dir, flatten("commit", "-m", commit_message, "--", getFilePaths(d))...); err != nil {
		return err
	}

	if _, err := gitCommand(checkout_dir, "push", repo, fmt.Sprintf("HEAD:%s", branch)); err != nil {
		return err
	}

	fileListJson, err := json.Marshal(getFilePaths(d))
	if err != nil {
		return err
	}
	d.SetId(fmt.Sprintf("%s %s %s", repo, branch, string(fileListJson)))

	return nil
}
func CommitDelete(d *schema.ResourceData, meta interface{}) error {
	splits := strings.SplitN(d.Id(), " ", 3)
	repo := splits[0]
	branch := splits[1]

	workdir := meta.(*gitfileConfig).workDir
	checkout_dir := path.Join(workdir, mungeGitDir(d.Id()))
	commit_message := d.Get("commit_message").(string)

	if _, err := gitCommand(checkout_dir, flatten("rm", "--ignore-unmatch", "--", getFilePaths(d))...); err != nil {
		return err
	}

	if _, err := gitCommand(checkout_dir, flatten("diff-index", "--exit-code", "--quiet", "HEAD", "--", getFilePaths(d))...); err != nil {
		exitErr, isExitErr := err.(*exec.ExitError)
		if isExitErr {
			if exitErr.Sys().(syscall.WaitStatus).ExitStatus() != 1 {
				return err
			} else {
				if _, err := gitCommand(checkout_dir, flatten("commit", "-m", commit_message, "--", getFilePaths(d))...); err != nil {
					return err
				}

				if _, err := gitCommand(checkout_dir, "push", repo, fmt.Sprintf("HEAD:%s", branch)); err != nil {
					return err
				}
			}
		} else {
			return err
		}
	}

	return nil
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

func mungeGitDir(id string) string {
	return b64.URLEncoding.EncodeToString([]byte(id))
}

func shallowSparseGitCheckout(checkout_dir, repo, branch, bogus_file_name string, filepaths []string) error {
	if err := os.MkdirAll(checkout_dir, 0755); err != nil {
		return err
	}

	// git init appears to be idempotent.
	if _, err := gitCommand(checkout_dir, "init"); err != nil {
		return err
	}

	if _, err := gitCommand(checkout_dir, "config", "core.sparsecheckout", "true"); err != nil {
		return err
	}

	if err := ioutil.WriteFile(
		path.Join(checkout_dir, ".git", "info", "sparse-checkout"),
		[]byte(fmt.Sprintf("%s\n%s", bogus_file_name, strings.Join(filepaths, "\n"))),
		0666,
	); err != nil {
		return err
	}

	// if _, err := gitCommand(checkout_dir, "fetch", "--depth", "1", repo, branch); err != nil {
	if _, err := gitCommand(checkout_dir, "fetch", repo, branch); err != nil {
		return err
	}

	// I would have done "git checkout --force FETCH_HEAD" here, but if none of the files in filepaths
	// exist, then git fails with "error: Sparse checkout leaves no entry on working directory".
	// This set of steps works around that.
	if _, err := gitCommand(checkout_dir, "reset", "--soft", "FETCH_HEAD"); err != nil {
		return err
	}

	// doesn't matter that Create will truncate the file here; the read-tree command later will undo changes.
	if file, err := os.Create(path.Join(checkout_dir, bogus_file_name)); err != nil {
		return err
	} else {
		file.Close()
	}

	if _, err := gitCommand(checkout_dir, "add", bogus_file_name); err != nil {
		return err
	}

	if _, err := gitCommand(checkout_dir, "read-tree", "-u", "--reset", "HEAD"); err != nil {
		return err
	}

	return nil
}

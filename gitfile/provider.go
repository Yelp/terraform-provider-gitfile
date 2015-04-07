package gitfile

import (
	b64 "encoding/base64"
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


func commitResource() *schema.Resource {
	return &schema.Resource {
		Schema: map[string]*schema.Schema {
			"repo": &schema.Schema{
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
				Default: "Created by terraform gitfile_commit",
			},
			"file": &schema.Schema {
				Type: schema.TypeSet,
				Required: true,
				Set: hashFile,
				Elem: &schema.Resource{
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
				},
			},
		},
		Create: FileCreate,
		Read: FileRead,
		Update: FileUpdate,
		Delete: FileDelete,
	}
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

func FileCreate(d *schema.ResourceData, meta interface{}) error {
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
func FileRead(d *schema.ResourceData, meta interface{}) error {
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
func FileUpdate(d *schema.ResourceData, meta interface{}) error {
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
func FileDelete(d *schema.ResourceData, meta interface{}) error {
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
	out, err := command.Output()
	return out, err
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

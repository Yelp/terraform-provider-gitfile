package gitfile

import (
	"fmt"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"os/exec"
	"strings"
	"sync"
)

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

func hashString(v interface{}) int {
	switch v := v.(type) {
	default:
		panic(fmt.Sprintf("unexpectedtype %T", v))
	case string:
		return hashcode.String(v)
	}
}

// map of checkout_dir to lock. file, commit, and checkout should grab the lock corresponding to a checkout dir
// around create/read/update/delete operations.
var checkoutLocks map[string]*sync.Mutex

func lockCheckout(checkout_dir string) {
	if checkoutLocks == nil {
		checkoutLocks = map[string]*sync.Mutex{}
	}
	if checkoutLocks[checkout_dir] == nil {
		checkoutLocks[checkout_dir] = new(sync.Mutex)
	}
	checkoutLocks[checkout_dir].Lock()
}

func unlockCheckout(checkout_dir string) {
	checkoutLocks[checkout_dir].Unlock()
}

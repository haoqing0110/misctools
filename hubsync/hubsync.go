// Program hubsync pulls changes from a remote to the default branch
// of a local clone of a GitHub repository, then rebases local branches onto
// that branch if they have a corresponding remote branch.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"bitbucket.org/creachadair/shell"
)

var (
	defBranch    = flag.String("default", "", `Default branch name (if "", use default from remote)`)
	useRemote    = flag.String("remote", "origin", "Use this remote name")
	branchPrefix = flag.String("prefix", "", "Select branches matching this prefix")
	doForcePush  = flag.Bool("push", false, "Force push updated branches to remote")
	doVerbose    = flag.Bool("v", false, "Verbose logging")
)

func main() {
	flag.Parse()
	if *useRemote == "" {
		log.Fatal("You must specify a -remote name to use")
	}

	// Set working directory to the repository root.
	root, err := repoRoot()
	if err != nil {
		log.Fatalf("Repository root: %v", err)
	} else if err := os.Chdir(root); err != nil {
		log.Fatalf("Chdir: %v", err)
	}

	// Save the name of the current branch so we can go back.
	save, err := currentBranch()
	if err != nil {
		log.Fatalf("Current branch: %v", err)
	}

	// Find the name of the default branch.
	dbranch, err := defaultBranch(*defBranch, *useRemote)
	if err != nil {
		log.Fatalf("Default branch: %v", err)
	}
	defer func() {
		_, err := git("checkout", save)
		if err != nil {
			log.Fatalf("Switching to %q: %v", save, err)
		}
		log.Printf("Switched back to %q", save)
	}()

	// List local branches that track corresponding remote branches.
	rem, err := listBranchInfo(*branchPrefix+"*", dbranch, *useRemote)
	if err != nil {
		log.Fatalf("Listing branches: %v", err)
	}

	// Pull the latest content. Note we need to do this after checking branches,
	// since it changes which branches follow the default.
	log.Printf("Pulling default branch %q", dbranch)
	if err := pullBranch(dbranch); err != nil {
		log.Fatalf("Pull %q: %v", dbranch, err)
	}

	// Bail out if no branches need updating. But note we do this after pulling,
	// so that we will pull the changes even if no updates are required.
	if len(rem) == 0 {
		log.Print("No branches require update")
		return
	}

	// Rebase the local branches onto the default, and if requested and
	// necessary, push the results back up to the remote.
	for _, br := range rem {
		log.Printf("Rebasing %q onto %q", br.Name, dbranch)
		if _, err := git("rebase", dbranch, br.Name); err != nil {
			log.Fatalf("Rebase failed: %v", err)
		}
		if !*doForcePush || !br.Remote {
			continue
		}
		if ok, err := forcePush(*useRemote, br.Name); err != nil {
			log.Fatalf("Updating %q: %v", br.Name, err)
		} else if ok {
			log.Printf("- Forced update of %q to %s", br.Name, *useRemote)
		}
	}
}

func currentBranch() (string, error) { return git("branch", "--show-current") }

func forcePush(remote, branch string) (bool, error) {
	bits, err := exec.Command("git", "push", "-f", remote, branch).CombinedOutput()
	msg := string(bits)
	if err != nil {
		return false, errors.New(strings.SplitN(msg, "\n", 2)[0])
	}
	ok := strings.ToLower(strings.TrimSpace(msg)) != "everything up-to-date"
	return ok, nil
}

type branchInfo struct {
	Name   string
	Remote bool
}

func listBranchInfo(matching, dbranch, useRemote string) ([]*branchInfo, error) {
	var out []*branchInfo

	localOut, err := git("branch", "--list", "--contains", dbranch, matching)
	if err != nil {
		return nil, err
	}
	local := strings.Split(strings.TrimSpace(localOut), "\n")
	for _, s := range local {
		clean := strings.TrimPrefix(strings.TrimSpace(s), "* ")
		if clean == dbranch {
			continue
		}
		out = append(out, &branchInfo{Name: clean})
	}

	remoteOut, err := git("branch", "--list", "-r", useRemote+"/"+matching)
	if err != nil {
		return nil, err
	}
	remote := strings.Split(strings.TrimSpace(remoteOut), "\n")
	for i, s := range remote {
		remote[i] = strings.TrimPrefix(strings.TrimSpace(s), useRemote+"/")
	}

	for _, lb := range out {
		if lb.Name == dbranch {
			continue // don't consider the default branch regardless
		}
		for _, rb := range remote {
			if rb == lb.Name {
				lb.Remote = true
				break
			}
		}
	}
	return out, nil
}

func pullBranch(branch string) error {
	if _, err := git("checkout", branch); err != nil {
		return fmt.Errorf("checkout: %w", err)
	}
	_, err := git("pull")
	return err
}

func repoRoot() (string, error) { return git("rev-parse", "--show-toplevel") }

func defaultBranch(defBranch, useRemote string) (string, error) {
	if defBranch != "" {
		return defBranch, nil
	}
	rem, err := git("remote", "show", useRemote)
	if err != nil {
		return "", err
	}
	const needle = "HEAD branch:"
	for _, line := range strings.Split(rem, "\n") {
		i := strings.Index(line, needle)
		if i >= 0 {
			return strings.TrimSpace(line[i+len(needle):]), nil
		}
	}
	return "", errors.New("default branch not found")
}

func git(cmd string, args ...string) (string, error) {
	if *doVerbose {
		log.Println("[git]", cmd, shell.Join(args))
	}
	out, err := exec.Command("git", append([]string{cmd}, args...)...).Output()
	if err != nil {
		var ex *exec.ExitError
		if errors.As(err, &ex) {
			return "", errors.New(strings.SplitN(string(ex.Stderr), "\n", 2)[0])
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

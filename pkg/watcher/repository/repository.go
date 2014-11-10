package repository

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/openshift/source-to-image/pkg/sti/git"
	"github.com/openshift/source-to-image/pkg/sti/util"
)

type CommitUser struct {
	Name  string
	Email string
}

type CommitDetails struct {
	Commit    string
	Message   string
	Author    CommitUser
	Committer CommitUser
}

type RepositoryListener interface {
	CommitAvailable(info *CommitDetails)
}

type Watcher interface {
	Stop()
}

type repositoryWatcher struct {
	listener RepositoryListener
	uri      string
	ref      string
	git      git.Git
	runner   util.CommandRunner
	commit   string
	repoDir  string
	stop     bool
}

func WatchRepository(repositoryURI string, ref string, listener RepositoryListener, interval time.Duration) Watcher {
	w := &repositoryWatcher{
		uri:      repositoryURI,
		ref:      ref,
		listener: listener,
		git:      git.NewGit(),
		runner:   util.NewCommandRunner(),
	}
	go func() {
		t := time.Tick(interval)
		for _ = range t {
			w.watch()
			if w.stop {
				break
			}
		}
	}()
	return w
}

func (w *repositoryWatcher) watch() {
	if w.commit == "" {
		w.initializeRepo()
		return
	}
	w.checkRepoUpdate()
}

func (w *repositoryWatcher) initializeRepo() {
	log.Printf("Initializing repository: %s\n", w.uri)
	repoDir, err := ioutil.TempDir("", "watchrepo")
	if err != nil {
		log.Printf("Unable to intialize repository for %s, could not create local directory: %v", w.uri, err)
		return
	}
	w.repoDir = repoDir
	if err = w.git.Clone(w.uri, repoDir); err != nil {
		log.Printf("Unable to initialize repository for %s, could not clone it: %v", w.uri, err)
		return
	}
	if w.ref != "" {
		if err = w.git.Checkout(repoDir, w.ref); err != nil {
			log.Printf("Unable to initialize repository for %s, could not checkout ref %s: %v", w.uri, w.ref, err)
			return
		}
	}
	commitDetails, err := w.getCommitDetails()
	if err != nil {
		log.Printf("Unable to obtain commit information for %s: %v", w.uri, err)
		return
	}
	w.commit = commitDetails.Commit
	if w.listener != nil {
		w.listener.CommitAvailable(commitDetails)
	}
}

func (w *repositoryWatcher) checkRepoUpdate() {
	opts := util.CommandOpts{
		Dir: w.repoDir,
	}
	err := w.runner.RunWithOptions(opts, "git", "pull")
	if err != nil {
		log.Printf("Unable to check for updates. git pull failed: %v", err)
		return
	}
	commitDetails, err := w.getCommitDetails()
	if err != nil {
		log.Printf("Unable to check for updates. Could not retrieve commit details: %v", err)
	}
	if commitDetails.Commit != w.commit {
		w.commit = commitDetails.Commit
		w.listener.CommitAvailable(commitDetails)
	}
}

func (w *repositoryWatcher) getCommitDetails() (*CommitDetails, error) {
	buffer := bytes.Buffer{}
	opts := util.CommandOpts{
		Stdout: &buffer,
		Stderr: os.Stderr,
		Dir:    w.repoDir,
	}
	err := w.runner.RunWithOptions(opts, "git", "log", "--pretty=%H|%an|%ae|%cn|%ce|%s", "-n1")
	if err != nil {
		return nil, err
	}
	d := &CommitDetails{}
	parts := strings.Split(buffer.String(), "|")
	d.Commit = parts[0]
	d.Author.Name = parts[1]
	d.Author.Email = parts[2]
	d.Committer.Name = parts[3]
	d.Committer.Email = parts[4]
	d.Message = parts[5]
	return d, nil
}

func (w *repositoryWatcher) Stop() {
	w.stop = true
}

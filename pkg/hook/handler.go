package hook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/csrwng/gitpoll/pkg/watcher/buildcfg"
	"github.com/csrwng/gitpoll/pkg/watcher/repository"
	"github.com/openshift/origin/pkg/build/api"
)

type configWatcher struct {
	endpoint           string
	buildCfgInterval   time.Duration
	repositoryInterval time.Duration
	watchers           map[string]repository.Watcher
}

type buildLauncher struct {
	endpoint    string
	buildConfig *api.BuildConfig
}

func Start(endpoint string) {
	watcher := &configWatcher{
		endpoint:           endpoint,
		buildCfgInterval:   10 * time.Second,
		repositoryInterval: 10 * time.Second,
		watchers:           make(map[string]repository.Watcher),
	}
	watcher.run()
}

func (w *configWatcher) run() {
	buildcfg.WatchBuildConfigs(w.endpoint, w, w.buildCfgInterval)
}

func (w *configWatcher) BuildConfigAdded(bc *api.BuildConfig) {
	launcher := &buildLauncher{
		buildConfig: bc,
		endpoint:    w.endpoint,
	}
	watcher := repository.WatchRepository(bc.Parameters.Source.Git.URI, "", launcher, 10*time.Second)
	w.watchers[bc.ID] = watcher
}

func (w *configWatcher) BuildConfigDeleted(id string) {
	if watcher, ok := w.watchers[id]; ok {
		watcher.Stop()
		delete(w.watchers, id)
	}
}

type gitHubCommit struct {
	ID        string                `json:"id,omitempty" yaml:"id,omitempty"`
	Author    api.SourceControlUser `json:"author,omitempty" yaml:"author,omitempty"`
	Committer api.SourceControlUser `json:"committer,omitempty" yaml:"committer,omitempty"`
	Message   string                `json:"message,omitempty" yaml:"message,omitempty"`
}
type gitHubPushEvent struct {
	Ref        string       `json:"ref,omitempty" yaml:"ref,omitempty"`
	After      string       `json:"after,omitempty" yaml:"after,omitempty"`
	HeadCommit gitHubCommit `json:"head_commit,omitempty" yaml:"head_commit,omitempty"`
}

func (b *buildLauncher) CommitAvailable(info *repository.CommitDetails) {
	fmt.Printf("A commit is available: %v\n", info)
	e := gitHubPushEvent{
		Ref:   ref(b.buildConfig),
		After: info.Commit,
		HeadCommit: gitHubCommit{
			ID: info.Commit,
			Author: api.SourceControlUser{
				Name:  info.Author.Name,
				Email: info.Author.Email,
			},
			Committer: api.SourceControlUser{
				Name:  info.Committer.Name,
				Email: info.Committer.Email,
			},
			Message: info.Message,
		},
	}
	body, err := json.Marshal(e)
	if err != nil {
		log.Println("Unable to marshal git push event: %v", err)
		return
	}
	client := &http.Client{}
	url := webhookURL(b.endpoint, b.buildConfig)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		log.Println("Could not create webhook post request: %v", err)
		return
	}
	req.Header.Add("User-Agent", "GitHub-Hookshot/github")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Github-Event", "push")
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Webhook post request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		log.Println("An error occurred with the webhook request. Status: %s", resp.Status)
	}
}

func ref(bc *api.BuildConfig) string {
	refTag := "master"
	if bc.Parameters.Source.Git.Ref != "" {
		refTag = bc.Parameters.Source.Git.Ref
	}
	return "refs/heads/" + refTag
}

func webhookURL(endpoint string, bc *api.BuildConfig) string {
	return fmt.Sprintf("%s/osapi/v1beta1/buildConfigHooks/%s/%s/github",
		endpoint, bc.ID, bc.Secret)
}

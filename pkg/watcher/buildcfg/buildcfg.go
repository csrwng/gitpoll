package buildcfg

import (
	"fmt"
	"time"

	kapi "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	kclient "github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client/cache"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"

	oslatest "github.com/openshift/origin/pkg/api/latest"
	"github.com/openshift/origin/pkg/build/api"
	osclient "github.com/openshift/origin/pkg/client"
)

type Listener interface {
	BuildConfigAdded(bc *api.BuildConfig)
	BuildConfigDeleted(name string)
}

type Watcher interface {
	Stop()
}

type buildConfigWatcher struct {
	listener Listener
	store    cache.Store
	stop     bool
	client   *osclient.Client
	interval time.Duration
}

func WatchBuildConfigs(master string, listener Listener, interval time.Duration) (Watcher, error) {
	client, err := osclient.New(&kclient.Config{Host: master, Version: oslatest.Version})
	if err != nil {
		return nil, err
	}
	watcher := buildConfigWatcher{
		listener: listener,
		client:   client,
		interval: interval,
		store:    cache.NewStore(),
	}
	go func() { watcher.Watch() }()
	return &watcher, nil
}

func (w *buildConfigWatcher) Stop() {
	w.stop = true
}

func (w *buildConfigWatcher) Watch() {
	t := time.Tick(w.interval)
	for _ = range t {
		if w.stop {
			break
		}
		if err := w.sync(); err != nil {
			fmt.Errorf("An error occurred during sync: %v", err)
		}
	}
}

func (w *buildConfigWatcher) sync() error {
	buildConfigs, err := w.client.ListBuildConfigs(kapi.NewContext(), labels.Everything())
	if err != nil {
		return err
	}
	ids := w.store.Contains()
	for _, buildConfig := range buildConfigs.Items {
		if ids.Has(buildConfig.ID) {
			ids.Delete(buildConfig.ID)
		} else {
			w.store.Update(buildConfig.ID, buildConfig)
			w.listener.BuildConfigAdded(&buildConfig)
		}
	}

	for _, id := range ids.List() {
		w.store.Delete(id)
		w.listener.BuildConfigDeleted(id)
	}
	return nil
}

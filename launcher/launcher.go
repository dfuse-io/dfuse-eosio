// Copyright 2019 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package launcher

import (
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	pbdashboard "github.com/dfuse-io/dfuse-eosio/dashboard/pb"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type Launcher struct {
	*shutter.Shutter

	config  *DfuseConfig
	modules *RuntimeModules
	apps    map[string]App

	appStatus              map[string]pbdashboard.AppStatus
	appStatusSubscriptions []*subscription

	activeStatusLock     sync.RWMutex
	shutdownFatalLogOnce sync.Once
}

func NewLauncher(config *DfuseConfig, modules *RuntimeModules) *Launcher {
	l := &Launcher{
		Shutter:   shutter.New(),
		apps:      make(map[string]App),
		appStatus: make(map[string]pbdashboard.AppStatus),
		config:    config,
		modules:   modules,
	}
	// TODO: this is weird should re-think this? Should the launcher be passed in every Factory App func instead?
	// only the dashboard app that uses the launcher....
	l.modules.Launcher = l
	return l
}

func (l *Launcher) Launch(appNames []string) error {
	if len(appNames) == 0 {
		return fmt.Errorf("no apps specified")
	}
	// This is done first as a sanity check so we don't launch anything if something is misconfigured
	for _, appID := range appNames {
		appDef, found := AppRegistry[appID]
		if !found {
			return fmt.Errorf("cannot launch un-registered application %q", appID)
		}

		if appDef.InitFunc != nil {
			userLog.Debug("initialize application", zap.String("app", appID))
			err := appDef.InitFunc(l.modules)
			if err != nil {
				return fmt.Errorf("unable to initialize app %q: %w", appID, err)
			}
		}
	}

	for _, appID := range appNames {
		appDef := AppRegistry[appID]

		l.StoreAndStreamAppStatus(appID, pbdashboard.AppStatus_CREATED)
		userLog.Debug("creating application", zap.String("app", appID))
		app, err := appDef.FactoryFunc(l.modules)
		if err != nil {
			return fmt.Errorf("unable to create app %q: %w", appID, err)
		}
		l.OnTerminating(func(err error) {
			go app.Shutdown(err)
		})

		l.apps[appDef.ID] = app
	}

	for appID, app := range l.apps {
		if l.IsTerminating() {
			break
		}
		// run
		go func(appID string, app App) {
			defer (func() {
				// Don't be fooled, this will work only for this very goroutine and its
				// execution. If the app launches other goroutines and one of those fails, this
				// recovery will not be able to recover them and the whole program will panic
				// without ever reaching this point here.
				l.shutdownIfRecoveringFromPanic(appID, recover())
			})()

			userLog.Debug("launching app", zap.String("app", appID))
			err := app.Run()
			if err != nil {
				userLog.FatalAppError(appID, fmt.Errorf("unable to start: %w", err))
				l.Shutdown(nil)
			}
		}(appID, app)
	}

	for appID, app := range l.apps {
		if l.IsTerminating() {
			break
		}
		// watch for shutdown
		go func(appID string, app App) {
			select {
			case <-app.Terminating():
				userLog.Debug("app terminating", zap.String("app_id", appID))
				if err := app.Err(); err != nil {
					l.shutdownFatalLogOnce.Do(func() { // pretty printing of error causing dfuse shutdown
						userLog.FatalAppError(appID, err)
					})
					userLog.Error("app terminating with error", zap.String("app_id", appID), zap.Error(err))
				}
				l.StoreAndStreamAppStatus(appID, pbdashboard.AppStatus_STOPPED)

				l.Shutdown(nil)
			case <-l.Terminating():
			}
		}(appID, app)
	}

	// readiness probes
	go func() {
		for {
			allReady := l.updateReady()
			if allReady {
				time.Sleep(5 * time.Second)
			} else {
				time.Sleep(1 * time.Second)
			}
		}

	}()

	return nil
}

func (l *Launcher) shutdownIfRecoveringFromPanic(appID string, recovered interface{}) {
	if recovered == nil {
		return
	}

	err := fmt.Errorf("app %q panicked", appID)
	if recoveredErr, ok := recovered.(error); ok {
		err = fmt.Errorf("%s: %w\n%s", err.Error(), recoveredErr, string(debug.Stack()))
	}

	l.Shutdown(err)
}

func (l *Launcher) StoreAndStreamAppStatus(appID string, status pbdashboard.AppStatus) {
	l.activeStatusLock.Lock()
	defer l.activeStatusLock.Unlock()

	l.appStatus[appID] = status

	appInfo := &pbdashboard.AppInfo{
		Id:     appID,
		Status: status,
	}

	for _, sub := range l.appStatusSubscriptions {
		sub.Push(appInfo)
	}
}

func (l *Launcher) GetAppStatus(appID string) pbdashboard.AppStatus {
	l.activeStatusLock.RLock()
	defer l.activeStatusLock.RUnlock()

	if v, found := l.appStatus[appID]; found {
		return v
	}

	return pbdashboard.AppStatus_NOTFOUND
}

func (l *Launcher) GetAppIDs() (resp []string) {
	for appID, _ := range l.apps {
		resp = append(resp, string(appID))
	}
	return resp
}

func (l *Launcher) updateReady() (allReady bool) {
	allReady = true
	for appID, app := range l.apps {
		if readyableApp, ok := app.(readiable); ok {
			if readyableApp.IsReady() {

				if l.GetAppStatus(appID) != pbdashboard.AppStatus_RUNNING {
					userLog.Debug("app status switching to running", zap.String("app_id", appID))
					l.StoreAndStreamAppStatus(appID, pbdashboard.AppStatus_RUNNING)
				}
			} else {
				allReady = false
				if l.GetAppStatus(appID) != pbdashboard.AppStatus_WARNING {
					userLog.Debug("app status switching to warning", zap.String("app_id", appID))
					l.StoreAndStreamAppStatus(appID, pbdashboard.AppStatus_WARNING)
				}
			}
		} else {
			userLog.Debug("does not support readiness probe", zap.String("app_id", appID))
		}
	}
	return
}

func (l *Launcher) WaitForTermination() {
	userLog.Printf("Waiting for all apps termination...")
	now := time.Now()
	for appID, app := range l.apps {
	innerFor:
		for {
			select {
			case <-app.Terminated():
				userLog.Debug("App terminated", zap.String("app_id", appID))
				break innerFor
			case <-time.After(1500 * time.Millisecond):
				userLog.Printf("Still waiting for app %q ... %v", appID, time.Since(now).Round(100*time.Millisecond))
			}
		}
	}
	userLog.Printf("All apps terminated gracefully")
}

func (l *Launcher) SubscribeAppStatus() *subscription {
	chanSize := 500
	sub := newSubscription(chanSize)

	l.activeStatusLock.Lock()
	defer l.activeStatusLock.Unlock()

	l.appStatusSubscriptions = append(l.appStatusSubscriptions, sub)

	userLog.Debug("App status subscribed")
	return sub
}

func (l *Launcher) UnsubscribeAppStatus(sub *subscription) {
	if sub == nil {
		return
	}

	l.activeStatusLock.Lock()
	defer l.activeStatusLock.Unlock()

	var filtered []*subscription
	for _, candidate := range l.appStatusSubscriptions {
		// Pointer address comparison
		if candidate != sub {
			filtered = append(filtered, candidate)
		}
	}

	l.appStatusSubscriptions = filtered
}

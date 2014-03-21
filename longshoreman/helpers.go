// Helper methods for doing sequential or parallel
// docker API calls.
package longshoreman

import (
	"errors"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"strings"
	"time"
)

func (l *Longshoreman) sequentiallyCallForContainers(commandName string, cmdFunc dockerClientContainerCall) (errs []error) {
	cmdErrors := make([]error, 0)
	numHosts := len(l.Hosts)

	l.Logger.Info("Starting %s of %s on %d hosts", commandName, l.Image, numHosts)

	for _, host := range l.Hosts {
		client, err := docker.NewClient("http://" + host)
		if err != nil {
			cmdErrors = append(cmdErrors, err)
			continue
		}

		l.Logger.Debug("  - [%s] %s: %s", host, commandName, l.Image)
		containerIds, err := l.getContainerIds(host)
		if err != nil {
			cmdErrors = append(cmdErrors, err)
		}

		for _, cid := range containerIds {
			l.Logger.Debug("  - [%s] %s container %s", host, commandName, cid[:15])
			err = cmdFunc(client, l, host, cid)

			if err != nil {
				cmdErrors = append(cmdErrors, err)
			}
		}
	}

	l.Logger.Info("Completed %s of %s on %d hosts", commandName, l.Image, numHosts)

	return cmdErrors
}

func (l *Longshoreman) sequentiallyCallForHosts(commandName string, cmdFunc dockerClientHostCall) (errs []error) {
	cmdErrors := make([]error, 0)
	numHosts := len(l.Hosts)

	l.Logger.Info("Starting %s of %s on %d hosts", commandName, l.Image, numHosts)

	for _, host := range l.Hosts {
		client, err := docker.NewClient("http://" + host)
		if err != nil {
			cmdErrors = append(cmdErrors, err)
			continue
		}

		l.Logger.Debug("  - [%s] %s: %s", host, commandName, l.Image)
		err = cmdFunc(client, l)

		if err != nil {
			cmdErrors = append(cmdErrors, err)
		}
	}

	l.Logger.Info("Completed %s of %s on %d hosts", commandName, l.Image, numHosts)

	return cmdErrors
}

func (l *Longshoreman) parallelCallForHosts(commandName string, cmdFunc dockerAsyncClientHostCall) (errs []error) {
	doneChan := make(chan bool)
	errChan := make(chan RemoteError)
	numHosts := len(l.Hosts)
	completed := 0
	errored := 0
	cmdErrors := make([]RemoteError, 0)

	l.Logger.Info("Starting %s of %s on %d hosts", commandName, l.Image, numHosts)

	for _, host := range l.Hosts {
		client, err := docker.NewClient("http://" + host)
		if err != nil {
			cmdErrors = append(cmdErrors, RemoteError{err, host})
			continue
		}

		l.Logger.Debug("  - [%s] start %s: %s", host, commandName, l.Image)

		go func() {
			cmdFunc(client, l, host, doneChan, errChan)
			l.Logger.Debug("  - [%s] completed %s: %s", host, commandName, l.Image)
		}()
	}

	for completed < numHosts {
		select {
		case <-doneChan:
			completed += 1
		case err := <-errChan:
			completed += 1
			errored += 1
			cmdErrors = append(cmdErrors, err)
		case <-time.After(l.Config.PullTimeout):
			allErrors := make([]error, 0)
			for _, rerr := range cmdErrors {
				allErrors = append(allErrors, rerr.error)
			}
			return append(allErrors, errors.New(fmt.Sprintf("Timeout while waiting for parallel %s", commandName)))
		}

		l.Logger.Debug("  - %s status: %d of %d completed (%d completed in error)", commandName, completed, numHosts, errored)
	}

	l.Logger.Info("Completed %s of %s on %d hosts completed", commandName, l.Image, numHosts)
	allErrors := make([]error, 0)
	for idx, rerr := range cmdErrors {
		l.Logger.Error(" %s error %d: [%s] %s", commandName, idx+1, rerr.host, rerr.error)
		allErrors = append(allErrors, rerr.error)
	}

	return allErrors
}

func (l *Longshoreman) getContainerIds(host string) (containerIds []string, err error) {
	client, err := docker.NewClient("http://" + host)
	if err != nil {
		return containerIds, err
	}

	containers, err := client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return containerIds, err
	}

	for _, container := range containers {
		if l.shouldIncludeContainer(container.Image) {
			containerIds = append(containerIds, container.ID)
		}
	}

	return containerIds, err
}

func (l *Longshoreman) shouldIncludeContainer(containerImageName string) bool {
	parts := strings.Split(containerImageName, ":")
	if len(parts) == 3 {
		parts = []string{parts[0] + ":" + parts[1], parts[2]}
	}

	if parts[0] == l.Image {
		if l.ImageTag == "" {
			return parts[1] == "" || parts[1] == "latest"
		} else {
			return l.ImageTag == parts[1]
		}
	}
	return false
}

package longshoreman

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	// "io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

type Config struct {
	RestartTimeout time.Duration
	RepullTimeout  time.Duration
}

type Longshoreman struct {
	Hosts    []string
	Image    string
	ImageTag string
	Config   Config
	fakeIn   bytes.Buffer
	fakeOut  bytes.Buffer
	fakeErr  bytes.Buffer
}

type RemoteError struct {
	error error
	host  string
}

func New(hosts []string, image string) (l *Longshoreman) {
	l = new(Longshoreman)
	l.Hosts = hosts
	l.Image = image
	l.Config = Config{time.Duration(30 * time.Second), time.Duration(30 * time.Second)}
	return l
}

func (l *Longshoreman) pull(host string, name string, done chan bool, errors chan RemoteError) {
	log.Printf("  - [%s] Pull: %s", host, name)

	client, err := docker.NewClient("http://" + host)
	if err != nil {
		errors <- RemoteError{err, host}
		return
	}

	err = client.PullImage(docker.PullImageOptions{l.Image, "", &l.fakeOut}, docker.AuthConfiguration{})
	if err == nil {
		log.Printf("  - [%s] Pull: %s completed", host, name)
		done <- true
	} else {
		log.Printf("  - [%s] Pull: %s completed in error", host, name)
		errors <- RemoteError{err, host}
	}
}

func (l *Longshoreman) Repull() (errs []error) {
	doneChan := make(chan bool)
	errChan := make(chan RemoteError)
	numHosts := len(l.Hosts)

	log.Printf("Starting pull of %s on %d hosts\n", l.Image, numHosts)

	for _, host := range l.Hosts {
		go l.pull(host, l.Image, doneChan, errChan)
	}

	completed := 0
	errored := 0
	pullErrors := make([]RemoteError, 0)
	for completed < numHosts {
		select {
		case <-doneChan:
			completed += 1
		case err := <-errChan:
			completed += 1
			errored += 1
			pullErrors = append(pullErrors, err)
		case <-time.After(l.Config.RepullTimeout):
			allErrors := make([]error, 0)
			for _, rerr := range pullErrors {
				allErrors = append(allErrors, rerr.error)
			}
			return append(allErrors, errors.New("Timeout while waiting for parallel repull"))
		}

		log.Printf("  - status: %d of %d completed (%d completed in error)", completed, numHosts, errored)
	}

	log.Printf("Pull of %s on %d hosts completed\n", l.Image, numHosts)
	allErrors := make([]error, 0)
	for idx, rerr := range pullErrors {
		log.Printf(" Error %d: [%s] %s\n", idx+1, rerr.host, rerr.error)
		allErrors = append(allErrors, rerr.error)
	}

	return allErrors
}

func (l *Longshoreman) Restart() (errs []error) {
	restartErrors := make([]error, 0)
	numHosts := len(l.Hosts)

	log.Printf("Starting restart of %s on %d hosts\n", l.Image, numHosts)

	for _, host := range l.Hosts {
		client, err := docker.NewClient("http://" + host)
		if err != nil {
			restartErrors = append(restartErrors, err)
			continue
		}

		log.Printf("  - [%s] Restart: %s", host, l.Image)
		containerIds, err := getContainerIds(host, l.Image)
		if err != nil {
			restartErrors = append(restartErrors, err)
		}

		for _, cid := range containerIds {
			log.Printf("  - [%s] restarting container %s", host, cid[:15])
			err = client.RestartContainer(cid, uint(l.Config.RestartTimeout.Seconds()))

			if err != nil {
				restartErrors = append(restartErrors, err)
			}
		}
	}

	log.Printf("Completed restart of %s on %d hosts\n", l.Image, numHosts)

	return restartErrors
}

func getContainerIds(host string, name string) (containerIds []string, err error) {
	output, err := fetchJSON(fmt.Sprintf("http://%s/containers/json", host))
	if err != nil {
		return containerIds, err
	}

	for _, container := range output {
		if strings.Split(container["Image"].(string), ":")[0] == name {
			containerIds = append(containerIds, container["Id"].(string))
		}
	}

	return containerIds, err
}

func fetchJSON(url string) (output []map[string]interface{}, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return output, err
	}

	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&output)

	return output, err
}

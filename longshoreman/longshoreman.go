// Structs and public methods
//
package longshoreman

import (
	"archive/tar"
	"bytes"
	"github.com/fsouza/go-dockerclient"
	"io/ioutil"
	"time"
)

type Config struct {
	RestartTimeout time.Duration // Time to wait for container shutdown before killing
	PullTimeout    time.Duration // Total time to wait on pulls
	StopTimeout    time.Duration // Time to wait for each container to stop
}

type Longshoreman struct {
	Hosts    []string
	Image    string
	ImageTag string
	Config   Config
	Logger   *Logger
	fakeIn   bytes.Buffer
	fakeOut  bytes.Buffer
	fakeErr  bytes.Buffer
}

type RemoteError struct {
	error error
	host  string
}

type HostStatus struct {
	Host       string
	Containers []docker.Container
}

type HostContents struct {
	Host     string
	Contents string
}

func New(hosts []string, image string) (l *Longshoreman) {
	l = new(Longshoreman)
	l.Hosts = hosts
	l.Image = image
	l.Config = Config{
		time.Duration(10 * time.Second),
		time.Duration(30 * time.Second),
		time.Duration(10 * time.Second),
	}
	l.Logger = NewLogger(LogLevelInfo)
	return l
}

type dockerClientHostCall func(client *docker.Client, longshoreman *Longshoreman) error
type dockerAsyncClientHostCall func(client *docker.Client, longshoreman *Longshoreman, host string, done chan bool, errors chan RemoteError)
type dockerClientContainerCall func(client *docker.Client, longshoreman *Longshoreman, host string, containerId string) error

func (l *Longshoreman) Restart() (errs []error) {
	return l.sequentiallyCallForContainers("restart", func(client *docker.Client, longshoreman *Longshoreman, host string, containerId string) error {
		return client.RestartContainer(containerId, uint(longshoreman.Config.RestartTimeout.Seconds()))
	})
}

func (l *Longshoreman) Stop() (errs []error) {
	return l.sequentiallyCallForContainers("stop", func(client *docker.Client, longshoreman *Longshoreman, host string, containerId string) error {
		return client.StopContainer(containerId, uint(longshoreman.Config.RestartTimeout.Seconds()))
	})
}

func (l *Longshoreman) List(results chan HostStatus) (errs []error) {
	errs = l.parallelCallForHosts("list", func(client *docker.Client, longshoreman *Longshoreman, host string, done chan bool, errors chan RemoteError) {
		containers, err := client.ListContainers(docker.ListContainersOptions{})
		if err != nil {
			errors <- RemoteError{err, host}
			return
		}

		hostStatus := HostStatus{host, nil}
		hostStatus.Host = host
		hostStatus.Containers = make([]docker.Container, 0, len(containers))
		for _, container := range containers {
			if longshoreman.shouldIncludeContainer(container.Image) {
				containerDetail, err := client.InspectContainer(container.ID)
				if err != nil {
					errors <- RemoteError{err, host}
					return
				}

				hostStatus.Containers = append(hostStatus.Containers, *containerDetail)
			}
		}
		results <- hostStatus

		done <- true
	})

	close(results)

	return errs
}

func (l *Longshoreman) Cat(path string, results chan HostContents) (errs []error) {
	errs = l.sequentiallyCallForContainers("cat", func(client *docker.Client, longshoreman *Longshoreman, host string, containerId string) error {
		contents := HostContents{host, ""}
		buffer := new(bytes.Buffer)
		err := client.CopyFromContainer(docker.CopyFromContainerOptions{buffer, containerId, path})
		if err != nil {
			return err
		}

		// Extract the single file tar
		r := bytes.NewReader(buffer.Bytes())
		tr := tar.NewReader(r)
		_, err = tr.Next()
		if err != nil {
			return err
		}

		rawContents, err := ioutil.ReadAll(tr)
		contents.Contents = string(rawContents)

		results <- contents
		return err
	})

	close(results)
	return errs
}

func (l *Longshoreman) Pull() (errs []error) {
	return l.parallelCallForHosts("pull", func(client *docker.Client, longshoreman *Longshoreman, host string, done chan bool, errors chan RemoteError) {
		err := client.PullImage(docker.PullImageOptions{l.Image, "", &l.fakeOut}, docker.AuthConfiguration{})
		if err == nil {
			longshoreman.Logger.Debug("  - [%s] Pull: %s completed", host, longshoreman.Image)
			done <- true
		} else {
			longshoreman.Logger.Error("  - [%s] Pull: %s completed in error", host, longshoreman.Image)
			errors <- RemoteError{err, host}
		}
	})
}

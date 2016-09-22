package main

import (
	"os"
	dockerapi "github.com/fsouza/go-dockerclient"
	"fmt"
	"log"
	"strings"
	"bytes"
	"time"
)

func main() {
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		os.Setenv("DOCKER_HOST", "unix:///private/var/run/docker.sock")
	}

	docker, err := dockerapi.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	handler := &Handler{Client: docker}

	//listen to docker events
	listener := make(chan *dockerapi.APIEvents)
	if err = docker.AddEventListener(listener); err != nil {
		log.Fatal(err)
	}

	defer func() {
		err = docker.RemoveEventListener(listener)
		if err != nil {
			log.Fatal(err)
		}

	}()

	for msg := range listener {
		switch msg.Status {
		case "start":
			go handler.Add(msg.Actor.ID)
		}
	}

}

func WriteFile(client *dockerapi.Client, containerId, contents, filepath string, ) {
	opts := dockerapi.CreateExecOptions{
		AttachStderr: true,
		AttachStdin: true,
		AttachStdout: true,
		Cmd: []string{"bash", "-c", fmt.Sprintf("echo '%s' > %s", contents, filepath)},
		Container: containerId,
	}
	exec, err := client.CreateExec(opts)

	startOpts := dockerapi.StartExecOptions{
		OutputStream: os.Stdout,
		ErrorStream:  os.Stderr,
		InputStream:  os.Stdin,
		RawTerminal:  true,
	}

	client.StartExec(exec.ID, startOpts)

	if  err != nil {
		panic(err)
	}
}

type Handler struct {
	Client *dockerapi.Client
	Filepath string
}

func(h Handler) Add(containerId,  string)  {
	WriteFile(h.Client, containerId, "abc", h.Filepath)
}

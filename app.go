package main

import (
	"github.com/alecthomas/kingpin"
	dockerapi "github.com/fsouza/go-dockerclient"
	"github.com/skarnecki/docker-vault/handler"
	"log"
	"os"
)

const (
	OsxDockerSock     = "unix:///private/var/run/docker.sock"
	DefaultFilePath   = "/tmp/init-token"
	DefaultMappingKey = "secret/dockervault"
)

var (
	vaultToken   = kingpin.Flag("token", "Vault wrapped token").Required().String()
	filePath     = kingpin.Flag("filePath", "Path to inject Vault token in contaner filesystem").Default(DefaultFilePath).String()
	vaultAddress = kingpin.Flag("vault", "Valut address").Default("http://127.0.0.1:8200/").String()
	dockerHost   = kingpin.Flag("dockerHost", "Docker host address.").Default(OsxDockerSock).String()
	mappingKey   = kingpin.Flag("mappingKey", "Location of image -> policy mapping in vault").Default(DefaultMappingKey).String()
)

func main() {
	kingpin.Parse()
	os.Setenv("DOCKER_HOST", *dockerHost)

	docker, err := dockerapi.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	handle, err := handler.NewHandler(docker, *vaultAddress, *vaultToken, *filePath)

	if err != nil {
		log.Fatal(err)
	}

	//listen to docker events
	listener := make(chan *dockerapi.APIEvents)
	if err := docker.AddEventListener(listener); err != nil {
		log.Fatal(err)
	}

	defer func() {
		err = docker.RemoveEventListener(listener)
		if err != nil {
			log.Fatal(err)
		}

	}()

	err = handle.RefreshPolicies(*mappingKey)
	if err != nil {
		log.Fatal(err)
	}

	// token refresh loop
	go handler.RefreshLoop(handle, *mappingKey)

	// docker events loop
	for msg := range listener {
		switch msg.Status {
		case "start":
			//TODO skips
			value, err := handle.GetPolicyName(msg.Actor.Attributes["image"])
			if err != nil {
				//FIXME - just ignore?
				log.Fatalf("no mapping found for %s", msg.Actor.Attributes["image"])
			}
			err = handle.Add(msg.Actor.ID, value)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

}

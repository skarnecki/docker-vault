package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/alecthomas/kingpin"
	dockerapi "github.com/fsouza/go-dockerclient"
	"github.com/skarnecki/docker-vault/handler"
	"os"
	"strings"
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
	verbose      = kingpin.Flag("verbose", "Enable verbose logging").Bool()
)

func main() {
	kingpin.Parse()

	if *verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

	os.Setenv("DOCKER_HOST", *dockerHost)
	docker, err := dockerapi.NewClientFromEnv()
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Debug("Connected to Docker")

	handle, err := handler.NewHandler(docker, *vaultAddress, *vaultToken, *filePath)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Debug("Connected to Vault")

	//listen to docker events
	listener := make(chan *dockerapi.APIEvents)
	if err := docker.AddEventListener(listener); err != nil {
		logrus.Fatal(err)
	}

	defer func() {
		err = docker.RemoveEventListener(listener)
		if err != nil {
			logrus.Fatal(err)
		}

	}()

	err = handle.RefreshPolicies(*mappingKey)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Debug("Policies prefetch")

	// token refresh loop
	go handler.RefreshLoop(handle, *mappingKey)

	// docker events loop
	for msg := range listener {
		switch msg.Status {
		case "start":
			imageName := strings.Split(msg.Actor.Attributes["image"],":")[0]
			logrus.Debugf("Container started: %s [%s]", msg.Actor.ID, imageName)
			//TODO skips
			value, err := handle.GetPolicyName(imageName)
			if err != nil {
				logrus.Errorf("No policy mapping for image [%s]", imageName)
				break
			}
			logrus.Debugf("Policy found: %s -> %s", imageName, value)
			err = handle.Add(msg.Actor.ID, value)
			if err != nil {
				logrus.Error(err)
				break
			}
		}
	}

}

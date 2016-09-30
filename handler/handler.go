package handler

import (
	"fmt"
	dockerapi "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/vault/api"
	"os"
	"strings"

	"sync"
	"time"
	"github.com/Sirupsen/logrus"
	"errors"
)

type Handler struct {
	DockerClient  *dockerapi.Client
	VaultClient   *api.Client
	Filepath      string
	PolicyMapping map[string]interface{}
	mutex         sync.RWMutex
}

//Creates new handler
func NewHandler(docker *dockerapi.Client, vaultAddress, initToken, filePath string) (*Handler, error) {
	cfg := &api.Config{Address: vaultAddress}
	client, err := api.NewClient(cfg)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error creating vault client: %s", err.Error()))
	}
	sec, err := client.Logical().Unwrap(strings.TrimSpace(initToken))
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error unwrapping token: %s", err.Error()))
	}
	client.SetToken(sec.Auth.ClientToken)

	client.SetWrappingLookupFunc(wrapTokenCreation)
	return &Handler{
		DockerClient:  docker,
		VaultClient:   client,
		Filepath:      filePath,
		PolicyMapping: map[string]interface{}{},
	}, nil
}

//Add called when new container is created
func (h Handler) Add(containerId, kind string) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	h.VaultClient.Auth().Token().RenewSelf(0)
	token, err := h.VaultClient.Auth().Token().Create(&api.TokenCreateRequest{
		Policies: []string{kind},
	})
	if err != nil {
		return errors.New(fmt.Sprintf("Error creating token for %s[%s]: %s", containerId, kind, err.Error()))
	}
	return WriteFile(h.DockerClient, containerId, token.WrapInfo.Token, h.Filepath)
}

//RefreshToken refresh vault token
func (h Handler) RefreshToken() {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	h.VaultClient.Auth().Token().RenewSelf(0)
}

//GetPolicyName get policy by image
func (h Handler) GetPolicyName(imageName string) (string, error) {
	policy, ok := h.PolicyMapping[imageName]
	if !ok {
		return "", fmt.Errorf("no policy for %s found", imageName)
	}
	return policy.(string), nil
}

//RefreshPolicies refresh image -> policy mapping
func (h Handler) RefreshPolicies(key string) error {
	secret, err := h.VaultClient.Logical().Read(key)
	if err != nil || secret == nil {
		return errors.New(fmt.Sprintf("Error fetching policies from %s: %s", key, err.Error()))
	}

	h.mutex.Lock()
	defer h.mutex.Unlock()
	for k, v := range secret.Data {
		h.PolicyMapping[k] = v
	}
	return nil
}

//WriteFile write data to selected docker container
func WriteFile(client *dockerapi.Client, containerId, contents, filepath string) error {
	opts := dockerapi.CreateExecOptions{
		AttachStderr: true,
		AttachStdin:  true,
		AttachStdout: true,
		Cmd:          []string{"sh", "-c", fmt.Sprintf("echo '%s' > %s", contents, filepath)},
		Container:    containerId,
	}
	exec, err := client.CreateExec(opts)
	if err != nil {
		return errors.New(fmt.Sprintf("Error writing token to %s on path %s: %s", containerId, filepath, err.Error()))
	}

	startOpts := dockerapi.StartExecOptions{
		OutputStream: os.Stdout,
		ErrorStream:  os.Stderr,
		InputStream:  os.Stdin,
		RawTerminal:  true,
	}

	return client.StartExec(exec.ID, startOpts)
}

//RefreshLoop refresh token and policies
func RefreshLoop(handle *Handler, key string) {
	refresh := time.Tick(10 * time.Second)
	for range refresh {
		logrus.Debug("refreshed")
		handle.RefreshToken()
		handle.RefreshPolicies(key)
	}
}

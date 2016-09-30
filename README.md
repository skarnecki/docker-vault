# docker-vault

Inject Vault tokens to Docker containers!

Handles creating and injecting Vault tokens into spawned docker containers. It uses map to create token with proper permissions. Injected tokens are wrapped making whole process secure.

## Problem
When using docker containers created by scheduler, there is no safe way to manage Vault tokens. Docker vault tries to address that by providing separate process that generate tokens.

## Details

To start dockervault you need to provide wrapped token with permissions to generate tokens. It listens to container start events via Docker daemon API. When container is started, it will reach for Vault mapping entry (secret/dockervault). It's where relation imagename => policy is stored. Then new wrapped token is generated with proper policy and injected to container filesystem.

Options:
* --token Vault wrapped token used to generate application specific tokens
* --filePath Path on docker container filesystem where token should be placed
* --vault Address of vault server
* --dockerHost Docker server endpoint address
* --mappingKey Path to vault entry containing imageName to policy mapping.
* --verbose Set logger to debug level

## Usage

Best way to use dockervault is by spawning docker container. https://hub.docker.com/r/eskey/dockervault/

### Required Docker-Vault Token policy
```
 path "auth/token/*" {
   policy = "sudo"
 }
 path "secret/dockervault" { //Vault mapping entry
   policy = "read"
 }
```

### Vault mapping entry
By default it's set to secret/dockervault. It's a simple map where key is imagename and value name of the policy. 
```
vault write secret/dockervault  my.docker.registry.com/foo/bar=bar
```

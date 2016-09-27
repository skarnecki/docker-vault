# docker-vault

Inject Vault tokens to docker containers!

This service reacts on new docker container started. It generates new wrapped token with specified policy and inject it to freshly started docker container filesystem.

Options
--token
Vault wrapped token used to generate application specific tokens
--filePath 
Path on docker container filesystem where token should be placed
--vault
Address of vault server
--dockerHost
Docker server endpoint address
--mappingKey
Path to vault entry containing imageName to policy mapping.

Required Docker-Vault Token policy
 path "auth/token/*" {
   policy = "sudo"
 }
 path "secret/dockervault" {
   policy = "read"
 }

# Credder

a.k.a "credentialler"

use:

```
~ » credder                                                                                                                                                                                        stijn@wswolf11
Usage: gitlab-secrets [init|import|pull|push|diff|help] [OPTIONS]
Init: Pull using a project ID.
Pull: update the local file with the secrets from GitLab. Values will be unset.
Push: update the GitLab secrets with the local file. Will inject passwords.
Diff: show the difference between local and remote secrets.
Help: This message.
```

Helps you set up and manage gitlab credentials.

> Always be careful with credentials; do not push them.

All operations are safe, meaning they will ask for your input when changing things remotely (currently only `push`)

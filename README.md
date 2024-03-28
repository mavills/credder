# Credder

Work with remote variables in version control for CI/CD & visibility.

```
~ » credder                                                                                                                                                                                        stijn@wswolf11
Usage: gitlab-secrets [init|import|pull|apply|plan|help]
	init: Set up a new variable file.
	import: Overwrite local variables with remote.
	pull: Update local variables with remote.
	apply: Update remote variables with local.
	plan: Show staged local changes (what will change on GitLab).
	help: Show this message.
```

> Always be careful with credentials; do not push them.

All operations are safe, meaning they will ask for your input when changing things remotely (currently only `push`)

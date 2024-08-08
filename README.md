# GitHub Release Note Generator

This is a CLI tool generating GitHub release note from GitHub Projetcs.

# Usesage

```
Usage:
  ./gh-release-note-generator [flags]

Flags:
  -h, --help                  help for ./gh-release-note-generator
  -l, --labels stringArray    Target issue labels
  -i, --max-item-count int    Max item count in target Project (default 100)
  -o, --organization string   Organization
  -p, --project int           Target GitHub Project ID (default 1)
  -r, --repository string     Target repository
  -t, --token string          GitHub access token
  -v, --version               version for ./gh-release-note-generator
```

The permission the GitHub Access Token should hava to access `read:projects`.

```
export GITHUB_ACCESS_TOKEN="YOUR_GITHUB_ACCESS_TOKEN"
```

# Output format

```
## {Label Name}
* {Issue Title}
  * #1, #2, #3

## {Label Name}
* {Issue Title}
  * #4
* {Issue Title}
  * #5, #6
```

- `#Number` is a link of Pull Request associated with the issue.
- Output in the order of issue labels passed as arguments
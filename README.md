# go-repo-template

Repository [template](https://docs.github.com/en/repositories/creating-and-managing-repositories/creating-a-repository-from-a-template)
for creating new Go projects!

## Bootstrap

Click `Use this template` button and voila!
![2024-10-05_22-53](https://github.com/user-attachments/assets/ae397fc7-5fa5-49df-94c1-314572a223d8)

After you're done, run the interactive bootstrap CLI:

```shell
go run -C bootstrap .
```

The CLI will guide you through an interactive form to configure your new project:
- **GitHub Account Name**: The GitHub account or organization that owns this repository
- **Repository Name**: The name of your new repository
- **Include Binary Support?**: Whether to include goreleaser configuration and binary build workflows
- **Include Versioning Support?**: Whether to include release drafter and automated versioning workflows

## Devbox

This project utilizes [devbox](https://github.com/jetify-com/devbox) in order
to provide a consistent and reliable development environment.
You can however, If you choose so, install the required dependencies manually.

## Project structure

The template includes an example of
[recommended Go project layout](https://github.com/golang-standards/project-layout)
which includes `cmd`, `pkg` and `internal` directories.

## justfile

[justfile](https://github.com/casey/just) provides all the basic utilities
for the development workflow.
Feel free to extend it with additional recipes as you see fit.
The same justfile recipes are used in CI, this ensures consistent results
for both remote and local machines.

You can quickly inspect the recipes of justfile by running either:

```shell
just --list
# or simply
just
```

When writing new recipes, make sure you document them with a `#` comment
directly above the recipe, like so:

```just
# Document me!
new-recipe:
  echo "Hello"
```

## CI

Continuous integration pipelines utilize the same
[justfile](./justfile) recipes which
you run locally within reproducible `devbox` environment.
This ensures consistent behavior of the executed checks
and makes local debugging easier.

## Testing

You can run all unit tests with `just test`.
We also encourage inspecting test coverage during development, you can verify
if the paths you're interested in are covered with `just test-coverage`.

## Releasing binaries

If you decide to ship binaries with the project,
[Goreleaser](https://goreleaser.com/) will require setting up
`GORELEASER_TOKEN` secret.
Refer to [goreleaser-action](https://github.com/goreleaser/goreleaser-action)
docs for up-to-date permission requirements for the token.

## Release Drafter

If you decide to keep release automation, you will need to setup
`RELEASE_DRAFTER_TOKEN` secret.
Refer to [release-drafter](https://github.com/release-drafter/release-drafter?tab=readme-ov-file#usage)
docs for up-to-date permission requirements for the token.

## Labels

In order for some automations to work, like
[Release Drafter](https://github.com/release-drafter/release-drafter),
we need a predefined set of labels.
If you create a new repository from this template, the labels will be
automatically transferred for you.
However, if you want to use these automations in an existing repository,
you'll need to create these labels.
There's a convenience script for that,
located [here](./bootstrap/add-labels.bash).
Run the following:

```shell
./bootstrap/add-labels.bash <github-account-name> <repo-name>
```

If you wish to update existing labels, add `--force` to the `gh label create`
invocation in the script.

## Renovate

This template includes [Renovate](https://docs.renovatebot.com/) configuration
for automated dependency updates.
Renovate will automatically create pull requests to update your dependencies
when new versions are available.

The configuration is located in [.github/renovate.json5](./.github/renovate.json5).

To enable Renovate for your repository:

1. Install the [Renovate GitHub App](https://github.com/apps/renovate).
2. Grant it access to your repository.
3. Renovate will automatically start monitoring your dependencies.

## Gitsync

The author of this repository also uses it as a staple/root
for other repositories to follow.
This means things like linter configs or CI/CD workflows in these repositories
are supposed to be kept in sync with **this** repository (with some variations).

This is achieved with a tool called [gitsync](https://github.com/nieomylnieja/gitsync).
Configuration file for the tool is [gitsync.json](./gitsync.json).
The bootstrap script removes it.

In order to see the diff between managed repositories run:

```shell
gitsync -c gitsync.json diff
```

In order to sync the changes for managed repositories run:

```shell
gitsync -c gitsync.json sync
```

## License

The repository template comes with Mozilla Public License 2.0.
Feel free to change the license to any that suits you.

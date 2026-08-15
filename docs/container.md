# The container

For when you would rather not install anything at all.

```bash
docker run --rm -v "$PWD:/work" ghcr.io/sergeylubivui-dev/devtree render
docker run --rm -v "$PWD:/work" ghcr.io/sergeylubivui-dev/devtree board
docker run --rm -v "$PWD:/work" ghcr.io/sergeylubivui-dev/devtree add "Search filters" -p mvp
```

`/work` is where the repository gets mounted. Nothing is baked into the image, and the container
writes only what devtree writes: the plan file and the outputs it names.

## The editor, from the container

```bash
docker run --rm -p 9312:9312 -v "$PWD:/work" \
  ghcr.io/sergeylubivui-dev/devtree serve --host 0.0.0.0
```

Then open <http://127.0.0.1:9312/>.

`--host 0.0.0.0` is not optional here, and it is worth knowing why. The editor binds to loopback by
default, and inside a container loopback is the *container's* — a published port would arrive at an
interface nobody is listening on. Binding to every interface is what makes the published port reach
it.

That also means the usual protection is gone: the editor has no authentication, so anything that can
reach the port can edit the plan. Published as `-p 9312:9312` it is reachable from your network, not
just your machine. Bind it to your own address if you want the container's behaviour and loopback's
reach:

```bash
docker run --rm -p 127.0.0.1:9312:9312 -v "$PWD:/work" \
  ghcr.io/sergeylubivui-dev/devtree serve --host 0.0.0.0
```

devtree prints a warning whenever it is bound anywhere but loopback, so the terminal always says
which of the two you got.

## Tags

| Tag | Follows |
|---|---|
| `latest` | the most recent release |
| `0.2.0`, `0.3.0`, … | that release exactly |
| `edge` | `main` |

Published for `linux/amd64` and `linux/arm64`.

## File ownership

On Linux, a container writing into a bind mount creates files owned by root unless told otherwise:

```bash
docker run --rm -u "$(id -u):$(id -g)" -v "$PWD:/work" ghcr.io/sergeylubivui-dev/devtree render
```

An alias makes the whole thing disappear into the background:

```bash
alias devtree='docker run --rm -u "$(id -u):$(id -g)" -v "$PWD:/work" ghcr.io/sergeylubivui-dev/devtree'

devtree board
devtree add "Refunds" -p billing
```

## In CI, without a Go toolchain

```yaml
- name: Check the plan and the diagram
  run: |
    docker run --rm -v "$PWD:/work" ghcr.io/sergeylubivui-dev/devtree check --strict
    docker run --rm -u "$(id -u):$(id -g)" -v "$PWD:/work" ghcr.io/sergeylubivui-dev/devtree render
    git diff --exit-code
```

On GitLab, `devtree install gitlab` writes exactly that job for you — see
[automation](automation.md).

## Why Alpine and not scratch

A `scratch` image would be about three megabytes instead of twelve, and every devtree command would
work in it — except one. `devtree sync` reads what git already knows about merged branches, and a
scratch image has no git to read it with. The extra layer buys back the command that would otherwise
be missing.

The image also sets `safe.directory` globally, because a bind-mounted repository looks to git like a
directory owned by somebody else, and git refuses to touch those.

## Building it yourself

```bash
docker build -t devtree --build-arg VERSION=dev .
docker run --rm -v "$PWD:/work" devtree version
```

The Dockerfile cross-compiles rather than emulating: the compiler stage is pinned to
`$BUILDPLATFORM` and told what to produce through `$TARGETOS` and `$TARGETARCH`. Go cross-compiles
in seconds; emulating an arm64 toolchain under QEMU to reach the same binary takes minutes. Only the
final stage needs emulation, and all it does is add git.

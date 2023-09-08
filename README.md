<div align="center">
    <img src="etcbuilder.svg" alt="EtcBuilder logo" width="200">
    <p>EtcBuilder is a tool to generate an `/etc` path based on multiple etc states.</p>
</div>

## Usage

### CLI

```bash
Usage:
  EtcBuilder [command]

Available Commands:
  build       Build a etc overlay based on the given System and User etc
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command

Flags:
  -h, --help   help for EtcBuilder

Use "EtcBuilder [command] --help" for more information about a command.
```

### Library

Assuming we have the following directory structure:

- `/etc` - The current system etc
- `/update/etc` - The etc that should be applied to the system etc, i.e. the etc coming from an update
- `/user/changes/etc` - The user changes to the system etc

We can use the following code to generate the final etc:

```go

import (
    EtcBuilder "github.com/linux-immutability-tools/EtcBuilder/cmd"
)

func main() {
    err := EtcBuilder.ExtBuildCommand("/etc", "/update/etc", "/user/changes/etc")
    if err != nil {
        panic(err)
    }
}
```


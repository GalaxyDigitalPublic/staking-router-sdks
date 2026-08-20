# Staking Router SDKs

Generated client SDKs for the Alluvial Staking Router API.

**This repository is generated. Do not edit by hand** — changes are overwritten on
the next release. The sources live in `AlluvialFinance/staking-router`.

| Path | Contents |
| --- | --- |
| `go/rest/` | Go REST client, generated with [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) |
| `go/grpc/` | Go gRPC client, generated with [buf](https://buf.build) |
| `openapi/` | OpenAPI 3.0 specification for the public API |

Each Go module lives in its own subdirectory and is versioned independently via a
subdirectory tag.

## Go — REST client

```bash
go get github.com/GalaxyDigitalPublic/staking-router-sdks/go/rest
```

```go
import stakingclient "github.com/GalaxyDigitalPublic/staking-router-sdks/go/rest"

client, err := stakingclient.NewClientWithResponses("https://api.alluvial.finance")
```

Resolved via `go/rest/vX.Y.Z` tags.

## Go — gRPC client

```bash
go get github.com/GalaxyDigitalPublic/staking-router-sdks/go/grpc
```

```go
import stakingrouter "github.com/GalaxyDigitalPublic/staking-router-sdks/go/grpc"

client, err := stakingrouter.New(
	"api.staking.alluvial.finance:443",
	stakingrouter.WithToken("<ACCESS_TOKEN>"),
)
if err != nil {
	return err
}
defer client.Close()
```

Resolved via `go/grpc/vX.Y.Z` tags.

Both SDKs track the Staking Router version they were generated from.

Current version: `main-d969880`

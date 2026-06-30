# uigraph-mcp

[![license](https://img.shields.io/badge/license-BUSL--1.1-blue)](LICENSE)

[Model Context Protocol](https://modelcontextprotocol.io/) server for [UiGraph](https://github.com/uigraph-oss). Exposes UiGraph architecture data — services, diagrams, maps, schemas, and folders — as MCP tools so AI assistants can read and reason about your system context.

## Tools

| Tool | Description |
|---|---|
| `get_service_context` | Rich context for a service (catalog, APIs, schemas, diagrams) |
| `list_services` | List services in an organization |
| `get_service` | Get a single service by ID or slug |
| `list_api_groups` | List API groups for a service |
| `get_api_spec` | Get the OpenAPI spec for an API group |
| `list_endpoints` | List endpoints in an API group |
| `list_folders` | List folder hierarchy |
| `list_diagrams` | List diagrams in a folder |
| `get_diagram` | Get diagram content and metadata |
| `list_maps` | List system maps |
| `get_map` | Get map frames and focal points |
| `list_service_dbs` | List database schemas attached to a service |
| `get_db_schema` | Get a database schema definition |

## Local development

```bash
go run ./cmd/mcp
```

Requires a running [uigraph-api](https://github.com/uigraph-oss/uigraph-api) instance.

| Variable | Default | Description |
|---|---|---|
| `UIGRAPH_API_URL` | — | Base URL of uigraph-api (required) |
| `UIGRAPH_FRONTEND_URL` | — | Base URL of the UIGraph frontend, used for the login broker (required) |
| `UIGRAPH_MCP_PUBLIC_URL` | — | Public base URL of this MCP server, used to build the auth callback (required) |
| `PORT` | `8080` | HTTP listen port |
| `MCP_SERVER_NAME` | `uigraph-mcp` | MCP server name |
| `MCP_SERVER_VERSION` | `0.1.0` | MCP server version |

For hot reload during development, see `Dockerfile.dev` and `.air.toml`.

## Login broker

The server brokers user login so the CLI client only needs the MCP server URL.

| Endpoint | Description |
|---|---|
| `GET /auth/login?redirect_uri=&state=` | Stores the CLI callback + state, redirects the browser to `{UIGRAPH_FRONTEND_URL}/authorize` |
| `GET /auth/callback?token=&state=` | Receives the token from the frontend and redirects back to the CLI's `redirect_uri?token=&state=` |

Flow: `CLI → /auth/login → frontend /authorize → (user logs in) → /auth/callback → CLI`. The CLI then sends the returned token as `Authorization: Bearer`, which the server forwards to uigraph-api. Service accounts skip this and pass their `uig_…` token directly.

## Testing

```bash
go test ./... -race
```

## License

This project is licensed under the [Business Source License 1.1](LICENSE) (BUSL-1.1).

- **Source available today** — you can read, modify, and redistribute the code under the terms of the license.
- **Non-production use** — free for development, testing, evaluation, and internal proof-of-concept.
- **Production use** — requires a commercial license from UiGraph. Production use means any use that supports the ongoing operation of your business or organization.
- **Future open source** — each version automatically converts to [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0) four years after it is first published under BUSL.

BUSL is not an OSI-approved open source license during the initial term. For commercial licensing questions, open an issue or contact the maintainers.

## Related projects

- [uigraph-api](https://github.com/uigraph-oss/uigraph-api) — backend API
- [uigraph-ui](https://github.com/uigraph-oss/uigraph-ui) — web application
- [uigraph-graphql](https://github.com/uigraph-oss/uigraph-graphql) — GraphQL BFF
- [uigraph-gateway](https://github.com/uigraph-oss/uigraph-gateway) — CLI sync API
- [uigraph-sdk](https://github.com/uigraph-oss/uigraph-sdk) — TypeScript SDK
- [uigraph-deploy](https://github.com/uigraph-oss/uigraph-deploy) — self-hosted deployment
- [uigraph-scripts](https://github.com/uigraph-oss/uigraph-scripts) — database seed utilities

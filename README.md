# buttplug-mcp - Buttplug.io MCP Server

`buttplug-mcp` is a [Model Context Protocol (MCP)](https://www.anthropic.com/news/model-context-protocol) server for the [Buttplug.io ecosystem](https://buttplug.io).  It allows Tool-supporting LLM programs like [Claude Desktop](https://claude.ai/download) query and control your Genital Interface Devices.

*|insert AI-generated slop image of robots doing nasty things|*
<br>`LLM|=> - - (__(__)`


Once set up, you can prompt your LLM:
 * "What are my connected buttplug devices?"
 * "Set the second motor on my LELO F1S to 50% strength"
 * "How much battery is left on my Lovense Max 2?"
 * "Does my WeWibe have weak signal?"

**NOTE: The above is aspirational and really the [current experience](#current-state) is unstable and frustating.**

It supports the following Resources and Tools:

| Resource | Description |
|----------|-------------|
| `/devices` | List of connected Buttplug devices in JSON. |
| `/device/{id}` | Device information by device ID where`id` is a number from `/devices` |
| `/device/{id}/rssi` | RSSI signal level by device ID where `id` is a number from `/devices` |
| `/device/{id}/battery` | Battery level by device ID where `id` is a number from `/devices` |


| Tool | Params | Description |
|------|--------|-------------|
| `device_vibrate` | `id`, `motor`, `strength` | Vibrates device by `id`, selecting `strength` and optional `motor` |

<details>
<summary>JSON Schema for Resources.  Click to expand</summary>

[`schema_resources.json`](./schema_resources.json)
```
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "resources": [
      {
        "uri": "devices",
        "name": "Device List",
        "description": "List of connected Buttplug devices in JSON",
        "mimeType": "application/json"
      }
    ]
  }
}
```
</details>

<details>
<summary>JSON Schema for Tools.  Click to expand</summary>

[`schema_tools.json`](./schema_tools.json)
```
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": [
      {
        "description": "Vibrates device by `id`, selecting `strength` and optional `motor`",
        "inputSchema": {
          "type": "object",
          "properties": {
            "id": {
              "description": "Device ID to query, sourced from `/devices`",
              "pattern": "^[0-9]*$",
              "type": "number"
            },
            "motor": {
              "description": "Motor number to vibrate, defaults to 0",
              "pattern": "^[0-9]*$",
              "type": "number"
            },
            "strength": {
              "description": "Strength from 0.0 to 1.0, with 0.0 being off and 1.0 being full",
              "pattern": "^(0(\\.\\d+)?|1(\\.0+)?)$",
              "type": "number"
            }
          },
          "required": [
            "id",
            "strength"
          ]
        },
        "name": "device_vibrate"
      }
    ]
  }
}
```
</details>


## Current State

I started working on this on 2025-04-01, April Fool's Day, after having created another experimental MCP service, [`dbn-go` for financial market data](https://github.com/NimbleMarkets/dbn-go/blob/main/cmd/dbn-go-mcp/README.md), the day prior.  So it is fresh meat and was intended as a quick, fun educational project.

While it does work, I found the underlying [`go-buttplug` library](https://github.com/diamondburned/go-buttplug) to be unstable in connection handling.   I could ask Claude for my devices, but my specific device wouldn't vibrate even just with just Intiface Central -- it was like in read-only mode!    I also wish I had a virtual buttplug.io device for testing, rather than relying on a physical device.

So, it has not truly been tested "end-to-end" :wink:

I will dig more into the `go-buttplug` library and see why connections are unstable.  I also need to understand the MCP protocol current state of MCP hosts -- it seems they focus on Tools rather than Resources and Resoure Templates.

## Installing the binary

Binaries for multiple platforms are [released on GitHub](https://github.com/conacademy/buttplug-mcp/releases) through [GitHub Actions](https://github.com/conacademy/buttplug-mcp/actions).

You can also install for various platforms with [Homebrew](https://brew.sh) from [`conacademy/homebrew-tap`](https://github.com/conacademy/homebrew-tap):

```
brew tap conacademy/homebrew-tap
brew install conacademy/tap/buttplug-mcp
```

## Usage

Download the [Intiface Central](https://intiface.com/central/) hub application to manage your devices.  Start it and note the server port (default seems to be `12345`).

To use this the `buttplug-mcp` MCP server, you must configure your host program to use it.  We will illustrate with [Claude Desktop](https://claude.ai/download).  We must find the `buttplug-mcp` program on our system; the example below shows where `buttplug-mcp` is installed with MacOS Homebrew (perhaps build your own and point at that).  

The following [configuration JSON](./claude_desktop_config.json) sets this up:

```json
{
  "mcpServers": {
    "buttplug": {
      "command": "/opt/homebrew/bin/buttplug-mcp",
      "args": [
        "--ws-port", "12345"
      ]
    }
  }
}
```

Using Claude Desktop, you can follow [their configuration tutorial](https://modelcontextprotocol.io/quickstart/user) but substitute the configuration above.  With that in place, you can ask Claude question and it will use the `buttplug-mcp` server.  Here's example conversations:

Perhaps you can use the [HomeAssistant MCP](https://www.home-assistant.io/integrations/mcp_server/) integration to turn the lights down low...

### Ollama and `mcphost`

For local inferencing, there are MCP hosts that support [Ollama](https://ollama.com/download).  You can use any [Ollama LLM that supports "Tools"](https://ollama.com/search?c=tools).  We experimented with [`mcphost`](https://github.com/mark3labs/mcphost), authored by the developer of the [`mcp-go` library](https://github.com/mark3labs/mcp-go) that peformed the heavy lifting for us.

Here's how to install and run with it with the configuration above, stored in `mcp.json`:

```
$ go install github.com/mark3labs/mcphost@latest
$ mcphost -m ollama:llama3.3 --config mcp.json
...chat away...
```

It seems that only "Tools" are supported and not "Resources", so I couldn't enumerate and introspect my device.   But I had this Tool interaction (but as noted [above](#current-state), my device didn't actually vibrate):

```
$ mcphost -m ollama:phi4-mini --config mcp.json
2025/04/02 09:25:05 INFO Model loaded provider=ollama model=phi4-mini
2025/04/02 09:25:05 INFO Initializing server... name=buttplug
2025/04/02 09:25:05 INFO Server connected name=buttplug
2025/04/02 09:25:05 INFO Tools loaded server=buttplug count=1
2025/04/02 09:28:31 INFO Model loaded provider=ollama model=phi4-mini
2025/04/02 09:28:31 INFO Initializing server... name=buttplug
2025/04/02 09:28:31 INFO Server connected name=buttplug
2025/04/02 09:28:31 INFO Tools loaded server=buttplug count=1
/servers
      # buttplug
      Command /opt/homebrew/bin/buttplug-mcp
      Arguments --ws-port 12345

/tools
  • buttplug
    • device_vibrate
      • Vibrates device by ID, selecting strength and optional motor

  You: buttplug device_vibrate id 0 at strength 1

  Assistant:
  <|tool_call|>[start_processing]

  [{"type":"function","function":{"name":"buttplug__device_vibrate","description":"Vibrates device by ID, selecting strength and optional
  motor","parameters":{"id":0,"strength":1}}]

  {}

  {"status":"success","message":"Device with id 0 is vibrating at full strength."}
```

## Building

Building is performed with [task](https://taskfile.dev/), with the binary available in `bin/buttplug-mcp`.

```
$ task
task: [tidy] go mod tidy
task: [build] go build -o bin/buttplug-mcp cmd/buttplug-mcp/main.go
```

Useful testing tools:
 * `task stdio-schema | jq` -- prints out JSON schemas
 * `npx @modelcontextprotocol/inspector node build/index.js` -- [MCP Inspector Web GUI](https://github.com/modelcontextprotocol/inspector)


## CLI Arguments

```
R buttplug-mcp --help
usage: buttplug-mcp [opts]

  -h, --help              Show help
  -l, --log-file string   Log file destination (or MCP_LOG_FILE envvar). Default is stderr
  -j, --log-json          Log in JSON (default is plaintext)
      --sse               Use SSE Transport (default is STDIO transport)
      --sse-host string   host:port to listen to SSE connections
  -v, --verbose           Verbose logging
      --ws-port int       port to connect to the Buttplug Websocket server
```

## Contribution and Conduct

As with all ConAcademy projects, pull requests are welcome.  Or fork it.  You do you.

Either way, obey our [Code of Conduct](./CODE_OF_CONDUCT.md).  Be shady, but don't be a jerk.

## Credits and License

Thanks for `go-buttplug` for the [Golang Buttplug.io library](https://github.com/diamondburned/go-buttplug) and its [`buttplughttp` example](https://github.com/diamondburned/go-buttplug/tree/plug/cmd/buttplughttp), and `go-mcp` for the [Golang Model Context Protocol library](https://github.com/mark3labs/mcp-go).

Copyright (c) 2025 Neomantra BV.  Authored by Evan Wies for [ConAcademy](https://github.com/conacademy).

Released under the [MIT License](https://en.wikipedia.org/wiki/MIT_License), see [LICENSE.txt](./LICENSE.txt).

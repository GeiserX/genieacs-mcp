# Resources and tools

| Type            | What for                                                                   | MCP URI / Tool id                |
|-----------------|----------------------------------------------------------------------------|----------------------------------|
| **Resources**   | Consume GenieACS data read-only                                            | `genieacs://device/{id}`<br>`genieacs://file/{name}`<br>`genieacs://tasks/{id}`<br>`genieacs://devices/list`<br>`genieacs://presets/list`<br>`genieacs://provisions/list`<br>`genieacs://faults/{id}` |
| **Tools**       | Invoke actions on a CPE through GenieACS                                   | `reboot_device`<br>`download_firmware`<br>`refresh_parameter`<br>`set_parameter`<br>`get_parameter`<br>`manage_preset`<br>`manage_provision`<br>`search_devices`<br>`tag_device`<br>`connection_request`<br>`delete_task`<br>`retry_task` |

Everything is exposed over a single JSON-RPC endpoint (`/mcp`).  
LLMs / Agents can: `initialize → readResource → listTools → callTool` … and so on.

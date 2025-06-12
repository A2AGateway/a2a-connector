# A2A Connector

Open source integration connector for the A2A Gateway ecosystem.

## Overview

The A2A Connector provides adapters for integrating with various enterprise systems including:

- Oracle databases
- SAP systems  
- Salesforce CRM
- Custom REST/SOAP APIs
- File-based integrations

## Features

- **Multi-adapter support** - Connect to different system types
- **Configuration-driven** - YAML-based setup
- **Proxy capabilities** - Transform data between systems
- **Extensible** - Add custom adapters

## Quick Start

```bash
go mod download
go run cmd/connector/main.go
```

## Configuration

See `config/` directory for example configurations:
- `example-banking.yaml`
- `example-crm.yaml` 
- `example-telecom.yaml`

## License

MIT License - see LICENSE file for details.

## Contributing

Contributions welcome! Please read our contributing guidelines.
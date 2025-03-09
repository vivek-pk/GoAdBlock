<div align="center">
<p><strong>A DNS-based ad blocker with a stylish dual-themed dashboard</strong></p>
</div>

<div align="center">
<p><em>⚠️ This is a work in progress application and may contain bugs or incomplete features ⚠️</em></p>
</div>

GoAdBlock is a lightweight, high-performance DNS-based ad blocker written in Go. It intercepts DNS queries for known advertising and tracking domains and prevents them from resolving, effectively blocking ads at the network level before they're downloaded.

## ✨ Features

- DNS-level ad blocking: Blocks ads at the network level for all devices
- Dual-themed dashboard: Choose between TVA (Time Variance Authority) or Cockpit interface
- Real-time statistics: Monitor blocked requests, cache performance, and more
- Client tracking: See which devices are making requests on your network
- Performance optimized: Written in Go for high throughput and low resource usage
- Self-contained binary: Single binary that includes all assets
- Local caching: Improves response times for frequently accessed domains
- Customizable blocklists: Add or remove domains from blocklists
- Cross-platform: Works on Linux, macOS, and Windows

## 📸 Screenshots

<div align="center">
<p><i>TVA Theme & Cockpit Theme</i></p>
</div>

## 🚀 Installation

### Prerequisites

- Go 1.18 or higher

### From Source

```sh
# Clone the repository
git clone https://github.com/vivek-pk/GoAdBlock.git

# Navigate to the project directory
cd GoAdBlock

# Build the project
go build -o goadblock ./cmd/server/main.go

# Run the executable
./goadblock
```

<!-- ### Using Docker
```sh
# Build the Docker image
docker build -t goadblock .

# Run the container
docker run -p 53:53/udp -p 8080:8080 goadblock
``` -->

## ⚙️ Configuration

> ⚠️ **TODO**: This section needs to be completed/reviewed

GoAdBlock can be configured using flags or a configuration file:

```sh
# Run with custom DNS port
./goadblock --dns-port=5353

# Run with custom web interface port
./goadblock --http-port=8080

# Use a config file
./goadblock --config=config.yaml
```

Example config file:

```yaml
dns:
  port: 53
  upstream: '8.8.8.8'
  cache_size: 5000
  cache_ttl: 3600

http:
  port: 8080
  username: 'admin'
  password: 'changeme'

blocklists:
  - 'https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts'
  - 'https://adaway.org/hosts.txt'
```

## 📊 Usage

1. Set your router's DNS server to point to the machine running GoAdBlock
2. Or configure individual devices to use GoAdBlock as their DNS server
3. Access the dashboard at http://<goadblock-ip>:8080
4. Toggle between themes using the theme switcher in the sidebar
5. Monitor blocking performance through the visual dashboard
6. Customize blocklists in the settings section

## 🛠️ Development

### Project Structure

```
/
├── cmd/
│   └── server/          # Application entry point
├── internal/
│   ├── api/                # Web API and dashboard
│   │   ├── static/         # Static assets (JS, CSS)
│   │   └── templates/      # HTML templates
│   ├── blocklist/          # Blocklist management
│   ├── cache/              # DNS cache implementation
│   ├── config/             # Configuration handling
│   └── dns/                # DNS server implementation
└── pkg/                    # Public packages
```

### Building for Development

```sh
# Run with hot reload
air -c .air.toml

# Build with debugging symbols
go build -gcflags=all="-N -l" -o goadblock ./cmd/goadblock
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Special thanks to everyone who contributed to this project
- UI themes inspired by Marvel's Time Variance Authority and aviation cockpit designs
- Built with Go, Alpine.js, Chart.js, and TailwindCSS

<div align="center">
<p>If you find this project useful, consider giving it a star! ⭐</p>
</div>

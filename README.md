<div align="center">
  <img src="images\captainPicard.png" alt="Application Logo"/>
  <h2>IDIOT</h2>
  <p><small><strong>Id</strong>entify <strong>I</strong>nternet <strong>O</strong>f <strong>T</strong>hings<br>A <em>GO</em> command line interface, that enables you to identify and manage internet of things (IOT) on your local network.</p>
  <p>
    <a href="https://github.com/bradleytenuta/idiot/actions"><img src="https://github.com/bradleytenuta/idiot/actions/workflows/test.yml/badge.svg" alt="Build Status"></a>
    <a href="https://codecov.io/gh/bradleytenuta/idiot"><img src="https://codecov.io/gh/bradleytenuta/idiot/branch/main/graph/badge.svg" alt="codecov"></a>
    <a href="https://github.com/bradleytenuta/idiot/releases/latest"><img src="https://img.shields.io/github/v/release/bradleytenuta/idiot?include_prereleases" alt="Latest Release"></a>
  </p>
  <img src="images\demo.gif" alt="Demo of application"/>
</div>

`idiot` is a command-line tool written in Go for discovering devices on your local network and connecting to them via SSH. It uses a combination of network protocols to build a comprehensive list of connected devices, enriches this data with hostnames and service information, and provides an easy way to initiate an SSH connection.

## How It Works

The tool discovers devices in a multi-phase process. First, it performs discovery to find live hosts, and then it enriches the data for those hosts.

1.  **Discovery Phase (Concurrent):**
    *   **ICMP Scan:** Pings every IP address in the local subnet to see who responds.
    *   **mDNS Scan:** Listens for devices announcing their services on the network.
2.  **Enrichment Phase (Concurrent):**
    *   **Reverse DNS Lookup:** Tries to find a hostname for the discovered IP addresses.
    *   **SSH Port Scan:** Checks if the standard SSH port (22) is open on each device.

This approach allows `idiot` to quickly build a detailed picture of your local network.

### Core Technologies Explained

Here’s a brief overview of the network protocols `idiot` uses, based on the implementation in the source code.

#### ICMP (Internet Control Message Protocol)
*   **Purpose:** To find active devices on the network that might not be advertising any services.
*   **How it's used (`internal/network/icmp.go`):** The tool iterates through all possible IP addresses on your local subnet (e.g., from `192.168.1.1` to `192.168.1.254`). For each address, it sends an `ICMP Echo Request` (a "ping"). Any device that responds with an `ICMP Echo Reply` is considered online and is added to the list of discovered devices. This is performed concurrently for speed.

#### mDNS (Multicast DNS)
*   **Purpose:** To discover services and devices on a local network without a central DNS server. It's how devices like Chromecasts, smart speakers, and printers announce themselves.
*   **How it's used (`internal/network/mdns.go`):** The application sends out a multicast query for `_services._dns-sd._udp`, which asks all mDNS-capable devices to report the services they offer. It then listens for responses, parsing them to extract IP addresses, hostnames, and sometimes even the device's model name (e.g., "Google Nest Mini").

#### DNS (Domain Name System)
*   **Purpose:** To resolve human-readable hostnames (like `my-laptop.local`) from IP addresses (`192.168.1.10`). This is also known as a "Reverse DNS Lookup".
*   **How it's used (`internal/network/dns.go`):** For each device found via ICMP, the tool performs a reverse DNS lookup. It asks the local network's DNS resolver (usually your router) if it has a name registered for that IP address. If a name is found, it's added to the device's details, making the list easier to read.

#### SSH (Secure Shell)
*   **Purpose:** To provide a secure way to remotely access and manage a device's command line.
*   **How it's used (`internal/network/ssh.go`, `cmd/ssh.go`):**
    1.  **Scanning:** During the enrichment phase, the tool attempts to open a TCP connection to port 22 (the default SSH port) on every discovered device. A successful connection indicates that an SSH server is likely running.
    2.  **Connecting:** When you run the `idiot ssh` command, it uses Go's `crypto/ssh` library to establish a full, interactive terminal session. It puts your local terminal into "raw mode" to ensure every keystroke is sent directly to the remote machine, allowing for a seamless remote shell experience.

## Developer Guide

This section contains instructions for developers who want to contribute to or build the project from the source.

### Prerequisites
*   Go (version 1.21 or newer)
*   golangci-lint for code linting

### Setup

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/bradleytenuta/idiot.git
    cd idiot
    ```
2.  **Install dependencies:**
    The project uses Go Modules. Dependencies will be downloaded automatically on the first build or can be fetched manually.
    ```bash
    go mod tidy
    ```

### Building from Source

To build the application into a single executable, run the following command from the root of the project. This will create an `idiot.exe` (on Windows) or `idiot` (on Linux/macOS) file in the project's root directory.

```bash
go build -o idiot ./src
```

### Running Tests

To run all unit tests for the project, execute the following command from the root directory:

```bash
go test ./...
```

To run tests with coverage:
```bash
go test -cover ./...
```

### Linting

We use `golangci-lint` to enforce code quality and style.

1.  **Install the linter:**
    ```bash
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    ```
2.  **Check for issues:**
    ```bash
    golangci-lint run
    ```
3.  **Automatically fix correctable issues:**
    ```bash
    golangci-lint run --fix
    ```

### Creating a Release

The project uses GitHub Actions to automatically create a release when a new version tag is pushed.

1.  **Ensure your `main` branch is up to date.**
2.  **Tag the release.** The version format must be `v[0-9]+\.[0-9]+\.[0-9]+`.
    ```bash
    git tag v0.0.1
    ```
3.  **Push the tags to trigger the action:**
    ```bash
    git push --tags
    ```
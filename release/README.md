# IDIOT (Identify Internet Of Things)

IDIOT is a command-line interface (CLI) tool built with Go that enables you to identify, manage, and interact with Internet of Things (IoT) devices on your local network.

## Features

*   **Network Discovery**: Scan your local network to find active devices using ICMP (ping) and mDNS protocols.
*   **Device Identification**: Gathers information like IPv4/IPv6 addresses, hostnames, and model names.
*   **SSH Connectivity**: Check for open SSH ports and launch an interactive SSH session directly to a discovered device.
*   **Device Persistence**: Save discovered devices to a configuration file for quick access later.
*   **Cross-Platform**: Runs on Windows, macOS, and Linux.

---

## Installation and Setup

To use the `idiot` CLI from anywhere in your terminal, you need to place the executable in a directory that is part of your system's `PATH` environment variable.

### 1. Get the Executable

First, ensure you have the `idiot` (or `idiot.exe` on Windows) executable file.

### 2. Add to PATH

Follow the instructions for your operating system.

#### Windows

1.  Create a folder where you want to store the executable, for example, `C:\Program-Files\idiot`.
2.  Move `idiot.exe` into this new folder.
3.  Press the `Windows Key`, type `env`, and select **"Edit the system environment variables"**.
4.  In the System Properties window, click the **"Environment Variables..."** button.
5.  In the "User variables" or "System variables" section, find and select the **`Path`** variable, then click **"Edit..."**.
6.  Click **"New"** and paste the full path to your folder (e.g., `C:\Program-Files\idiot`).
7.  Click **OK** on all open windows to save the changes.
8.  **Important**: Open a **new** Command Prompt or PowerShell window for the changes to take effect. You can now run `idiot --help`.

#### macOS & Linux

The recommended approach is to move the executable to `/usr/local/bin`, which is typically already in your `PATH`.

1.  Open your terminal.
2.  Move the `idiot` executable to `/usr/local/bin` and make it executable:
    ```sh
    # Move the file
    sudo mv /path/to/your/idiot /usr/local/bin/idiot

    # Make it executable
    sudo chmod +x /usr/local/bin/idiot
    ```
3.  You can now run `idiot --help` from any terminal window.

---

## Usage

The first time you run `idiot`, it will automatically create a `configuration.yaml` file in the same directory as the executable. This file is used to store saved devices and application settings.

### Global Flags

*   `--help`: Show help for any command.

### Commands

#### `scan`

Scans the local network to discover connected devices.

```sh
idiot scan
```

This command performs the following actions:
1.  Displays a spinner animation while it scans the network using ICMP and mDNS.
2.  Checks discovered devices for an open SSH port.
3.  Once the scan is complete, it presents you with an interactive list of all discovered devices.
4.  You can select a device from the list to save it to your `configuration.yaml` for future use with the `ssh` command.

---

#### `ssh`

Starts an interactive SSH session with a previously saved device.

```sh
idiot ssh
```

This command performs the following actions:
1.  Reads the list of saved devices from your `configuration.yaml` file.
2.  Presents you with an interactive list to choose which device you want to connect to.
3.  Prompts you to enter a **username** and **password** for the SSH session.
4.  Establishes the connection and gives you a remote shell on the device.

---

#### `version`

Prints the current version of the application.

```sh
idiot version
```

---

## Configuration

The `configuration.yaml` file is created automatically. You can edit it to manage your saved devices or to enable debug logging.

To enable more detailed output for troubleshooting, change the `debug` setting in the file:

```yaml
debug: true
```


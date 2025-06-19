## Useful Commands:

To create mod file for Go CLI application:

```bash
go mod init com.bradleytenuta/idiot
```

To build the Go CLI application into an executable.

```bash
go build .\main.go
```

The following commands that can be run on the executable as of now.

```bash
.\main.exe --help
.\main.exe version
```

## Downloading Dependencies:

Using Cobra and Viper as the main library to handle the CLI interface:

Cobra is a library for creating powerful modern CLI applications.

Link to Github: https://github.com/spf13/cobra

```bash
go get github.com/spf13/cobra@v1.9.1
```

Viper is a complete configuration solution for Go applications

```bash
go get github.com/spf13/viper@v1.20.1
```

Dependencies to scan local network to list all IP addresses currently connected to it.

```bash
go get github.com/hashicorp/mdns@v1.0.6
```
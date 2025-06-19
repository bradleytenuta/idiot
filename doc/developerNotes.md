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

Link to Github: https://github.com/spf13/cobra

```bash
go get -u github.com/spf13/cobra@latest
go get -u github.com/spf13/viper@latest
```
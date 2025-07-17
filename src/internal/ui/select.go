package ui

import (
	"errors"

	"com.bradleytenuta/idiot/internal/model"
	"github.com/manifoldco/promptui"
	"github.com/rs/zerolog/log"
)

// TODO: I see sometimes IPv6 is <nil> and not N/A
// TODO: Add a heading row like a table.
func CreateInteractiveSelect(iotDevices map[string]*model.Device, label string) (*model.Device, error) {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "▶ {{ .AddrV4 | cyan }}	{{ if ne .Hostname \"\" }}{{ .Hostname | magenta }}{{ end }}{{ if .CanConnectSSH }},{{ \"SSH OK\" | green }}{{ end }}",
		Inactive: "  {{ .AddrV4 | faint }}	{{ if ne .Hostname \"\" }}{{ .Hostname | magenta }}{{ end }}{{ if .CanConnectSSH }},{{ \"SSH OK\" | green }}{{ end }}",
		Selected: "✔ You selected {{ .AddrV4 | blue }} {{ if ne .Hostname \"\" }}{{ .Hostname | magenta }}{{ end }}{{ if .CanConnectSSH }},{{ \"SSH OK\" | green }}{{ end }}",
		Details: `
--------- Device Details ----------
{{ "IPv4 Address:" | faint }}	{{ .AddrV4 }}
{{ "IPv6 Address:" | faint }}	{{ if .AddrV6 }}{{ .AddrV6 }}{{ else }}N/A{{ end }}
{{ "MAC Address:" | faint }}	{{ if .MAC }}{{ .MAC }}{{ else }}N/A{{ end }}
{{ "Hostname:" | faint }}	{{ if ne .Hostname "" }}{{ .Hostname | magenta }}{{ else }}N/A{{ end }}
{{ "SSH Ready:" | faint }}	{{ if .CanConnectSSH }}{{ \"SSH OK\" | green }}{{ else }}N/A{{ end }}
{{ "Sources:" | faint }}	{{ .Sources }}`,
	}

	// Convert the map of discovered devices into a slice for the prompt.
	devices := make([]*model.Device, 0, len(iotDevices))
	for _, dev := range iotDevices {
		devices = append(devices, dev)
	}

	if len(devices) == 0 {
		return nil, errors.New("no IOT devices to select")
	}

	prompt := promptui.Select{
		Label:     label,
		Items:     devices,
		Templates: templates,
		// Display up to 10 items at once.
		Size: 10,
	}

	i, _, err := prompt.Run()
	if err != nil {
		// Handle user interruption (e.g., Ctrl+C).
		if err == promptui.ErrInterrupt {
			// TODO: Maybe should be using fmt to pass error message upstream?
			log.Error().Msg("Selection cancelled by user")
			return nil, err
		}
		log.Error().Msgf("Prompt failed %v", err)
		return nil, err
	}
	return devices[i], nil
}

func GetPromptInput(label string, mask rune) (string, error) {
	prompt := promptui.Prompt{
		Label: label,
	}
	if mask != 0 {
		prompt.Mask = mask
	}
	return prompt.Run()
}

package ui

import (
	"errors"

	"github.com/manifoldco/promptui"
	"github.com/rs/zerolog/log"

	"com.bradleytenuta/idiot/internal/model"
)

// CreateInteractiveSelect displays a list of discovered IoT devices to the user
// and allows them to select one. It uses promptui to create a rich, interactive list.
func CreateInteractiveSelect(iotDevices map[string]*model.Device) (*model.Device, error) {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "> {{ .AddrV4 | cyan }}\t{{ if .CanConnectSSH }}{{ \"SSH OK\" | green }}{{ end }}\t{{ if ne .Hostname \"\" }}{{ .Hostname | magenta }}{{ end }}",
		Inactive: "  {{ .AddrV4 | faint }}\t{{ if .CanConnectSSH }}{{ \"SSH OK\" | green }}{{ end }}\t{{ if ne .Hostname \"\" }}{{ .Hostname | magenta }}{{ end }}",
		Selected: "> You selected {{ .AddrV4 | blue }}{{ if .CanConnectSSH }} {{ \"SSH OK\" | green }}{{ end }}{{ if ne .Hostname \"\" }} {{ .Hostname | magenta }}{{ end }}",
		Details: `
Total IOT Devices found: {{ .Total }}
--------- Device Details ----------
{{ "IPv4 Address:" | faint }}	{{ .AddrV4 }}
{{ "IPv6 Address:" | faint }}	{{ if .AddrV6 }}{{ .AddrV6 }}{{ else }}N/A{{ end }}
{{ "MAC Address:" | faint }}	{{ if .MAC }}{{ .MAC }}{{ else }}N/A{{ end }}
{{ "Hostname:" | faint }}	{{ if ne .Hostname "" }}{{ .Hostname | magenta }}{{ else }}N/A{{ end }}
{{ "SSH Ready:" | faint }}	{{ if .CanConnectSSH }}{{ "SSH OK" | green }}{{ else }}N/A{{ end }}
{{ "Sources:" | faint }}	{{ .Sources }}`,
	}

	totalDevices := len(iotDevices)
	if totalDevices == 0 {
		return nil, errors.New("no IOT devices to select")
	}

	// Convert the map of discovered devices into a slice of our wrapper struct.
	selectItems := make([]model.SelectItem, 0, totalDevices)
	for _, dev := range iotDevices {
		selectItems = append(selectItems, model.SelectItem{Device: dev, Total: totalDevices})
	}

	prompt := promptui.Select{
		Label:     "    IPv4\t\tSSH\tHostname",
		Items:     selectItems,
		Templates: templates,
		Size:      10,
	}

	i, _, err := prompt.Run()
	if err != nil {
		// Handle user interruption (e.g., Ctrl+C).
		if err == promptui.ErrInterrupt {
			log.Debug().Msg("Selection cancelled by user")
			return nil, err
		}
		log.Error().Msgf("Prompt failed %v", err)
		return nil, err
	}
	return selectItems[i].Device, nil
}

// GetPromptInput displays a prompt to the user and returns the entered string.
// It can optionally mask the input, which is useful for passwords.
func GetPromptInput(label string, mask rune) (string, error) {
	prompt := promptui.Prompt{
		Label:       label,
		HideEntered: true,
	}
	if mask != 0 {
		prompt.Mask = mask
	}
	return prompt.Run()
}

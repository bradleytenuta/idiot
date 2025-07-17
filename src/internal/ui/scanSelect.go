package ui

import (
  "fmt"
  "log"
  "com.bradleytenuta/idiot/internal/model"
  "github.com/manifoldco/promptui"
)

// TODO: Update to provide label.
func CreateInteractiveSelect(discoveredDevices map[string]*model.Device) *model.Device {
	// Convert the map of discovered devices into a slice for the prompt.
  devices := make([]*model.Device, 0, len(discoveredDevices))
  for _, dev := range discoveredDevices {
    devices = append(devices, dev)
  }

  // If no devices were found, print a message and exit.
  if len(devices) == 0 {
    fmt.Println("No devices discovered on the network.")
    return nil
  }

  // Define custom templates for promptui to display device information nicely.
  templates := &promptui.SelectTemplates{
    Label:    "{{ . }}?",
    Active:   "▶ {{ .AddrV4 | cyan }} {{ if ne .Hostname \"\" }} ({{ .Hostname | green }}) {{ end }}",
    Inactive: "  {{ .AddrV4 | faint }} {{ if ne .Hostname \"\" }} ({{ .Hostname | faint }}) {{ end }}",
    Selected: "✔ You selected {{ .AddrV4 | green }}{{ if ne .Hostname \"\" }} ({{ .Hostname | green }}){{ end }}",
    Details: `
--------- Device Details ----------
{{ "IP Address:" | faint }}	{{ .AddrV4 }}
{{ "Hostname:" | faint }}	{{ if ne .Hostname "" }}{{ .Hostname }}{{ else }}N/A{{ end }}
{{ "MAC Address:" | faint }}	{{ if .MAC }}{{ .MAC }}{{ else }}N/A{{ end }}
{{ "SSH Ready:" | faint }}	{{ .CanConnectSSH }}
{{ "Sources:" | faint }}	{{ .Sources }}`,
  }

  // Create the interactive prompt for the user to select a device.
  prompt := promptui.Select{
    Label:     "Select a Discovered Device",
    Items:     devices,
    Templates: templates,
    Size:      10, // Display up to 10 items at once.
  }

  i, _, err := prompt.Run()
  if err != nil {
    // Handle user interruption (e.g., Ctrl+C).
    if err == promptui.ErrInterrupt {
      fmt.Println("Selection cancelled.")
      return nil
    }
    fmt.Printf("Prompt failed %v\n", err)
    return nil
  }

  selectedDevice := devices[i]

  fmt.Println("--- Selected Device ---")
  fmt.Printf("  IP Address:     %s\n", selectedDevice.AddrV4)
  fmt.Printf("  Hostname:       %s\n", selectedDevice.Hostname)
  fmt.Printf("  MAC Address:    %s\n", selectedDevice.MAC)
  fmt.Printf("  SSH Ready:      %t\n", selectedDevice.CanConnectSSH)
  fmt.Printf("  Discovered via: %v\n", selectedDevice.Sources)

  return selectedDevice
}

// TODO: Rename file or move.
func GetPromptInput(label string, mask rune) string {
	prompt := promptui.Prompt{
		Label: label,
	}
  if mask != 0 {
		prompt.Mask = mask
	}
	result, err := prompt.Run()
	if err != nil {
    // TODO: Throw error here.
    log.Fatalf("Prompt for '%s' failed: %v", label, err)
  }
  return result
}
package cmd

import (
	"sync"
  "com.bradleytenuta/idiot/internal"
	"com.bradleytenuta/idiot/internal/model"
	"com.bradleytenuta/idiot/internal/network"
	"com.bradleytenuta/idiot/internal/ui"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(scanCmd)
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan the local network and list devices connected to it.",
	Long:  `Scan the local network of this host and list the IP Addresses of devices connected to it. Including IPv4, IPv6 and if SSH is available.`,
	Run:   runScan,
}

func runScan(cmd *cobra.Command, args []string) {
	networkAddr, broadcastAddr, iface, err := network.GetInternetFacingNetworkInfo()
	if err != nil {
		log.Error().Msgf("Error setting up network: %v\n", err)
		return
	}

	discoveredDevices := make(map[string]*model.Device)
	var mu sync.Mutex

	network.PerformMdnsScan(iface, discoveredDevices, &mu)
	network.PerformIcmpScan(networkAddr, broadcastAddr, discoveredDevices, &mu)
	network.PerformSSHScan(discoveredDevices)
	network.PerformReverseDnsLookUp(discoveredDevices, &mu)

  cmd.Println("Select an IOT device to save for later use:")
	selectedIotDevice, _ := ui.CreateInteractiveSelect(discoveredDevices)
	if selectedIotDevice != nil {
		internal.SaveSelectedIotDevice(selectedIotDevice)
	} else {
		log.Debug().Msg("No device selected. Configuration not updated.")
	}
}
// Important note on ARP cache:
// After a successful ping, the device's MAC address should be in your system's ARP cache.
// Retrieving this from Go directly is OS-dependent and often involves:
// - Parsing `arp -a` output (less ideal for programmatic use)
// - Using a CGO binding to low-level network functions
// - Reading `/proc/net/arp` on Linux systems.
// The Go standard library does not provide a direct way to query the ARP cache.
// For a production-grade network scanner, you'd likely integrate with a library
// that wraps these OS-specific calls or use CGO.
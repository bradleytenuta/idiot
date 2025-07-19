package cmd

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"com.bradleytenuta/idiot/internal"
	"com.bradleytenuta/idiot/internal/model"
	"com.bradleytenuta/idiot/internal/network"
	"com.bradleytenuta/idiot/internal/ui"
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

	var mu sync.Mutex
	discoveredDevices := make(map[string]*model.Device)

	// Goroutine to display a spinner while scanning.
	done := make(chan bool)
	go func() {
		spinner := []string{"-", "\\", "|", "/"}
		i := 0
		for {
			select {
			case <-done:
				cmd.Print("\r \r") // Clear the spinner line
				return
			default:
				cmd.Printf("\rScanning for devices... %s ", spinner[i])
				i = (i + 1) % len(spinner)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	var wg sync.WaitGroup

	// Phase 1: Discover devices on the network.
	// These can run concurrently.
	wg.Add(2)
	go func() {
		defer wg.Done()
		network.PerformMdnsScan(iface, discoveredDevices, &mu)
	}()
	go func() {
		defer wg.Done()
		network.PerformIcmpScan(networkAddr, broadcastAddr, discoveredDevices, &mu)
	}()
	wg.Wait() // Wait for discovery to complete.

	// Phase 2: Enrich the discovered device data.
	// These can also run concurrently now that the initial list is populated.
	wg.Add(2)
	go func() {
		defer wg.Done()
		network.PerformSSHScan(discoveredDevices)
	}()
	go func() {
		defer wg.Done()
		network.PerformReverseDnsLookUp(discoveredDevices, &mu)
	}()
	wg.Wait() // Wait for enrichment to complete.

	close(done) // Stop the spinner.

	cmd.Println("\nSelect an IOT device to save for later use:")
	selectedIotDevice, _ := ui.CreateInteractiveSelect(discoveredDevices)
	if selectedIotDevice != nil {
		internal.SaveSelectedIotDevice(selectedIotDevice)
	} else {
		log.Debug().Msg("No device selected. Configuration not updated.")
	}
}

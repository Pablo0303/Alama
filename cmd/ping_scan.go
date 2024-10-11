package cmd

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/go-ping/ping"
	"github.com/spf13/cobra"
)

// pingScanCmd represents the pingScan command
var pingScanCmd = &cobra.Command{
	Use:   "ping",
	Short: "Ping a range of IPs or a list of IPs/hosts",
	Run:   pingScanRun,
}

var (
	pingFlagCIDR    string
	pingFlagFile    string
	pingFlagOutput  string
	pingFlagTimeout int
	pingFlagDelay   int
	pingFlagCount   int
	pingFlagThreads int
)

func init() {
	rootCmd.AddCommand(pingScanCmd)

	pingScanCmd.Flags().StringVarP(&pingFlagCIDR, "cidr", "c", "", "CIDR range to ping")
	pingScanCmd.Flags().StringVarP(&pingFlagFile, "file", "f", "", "File containing list of IPs/hosts to ping")
	pingScanCmd.Flags().StringVarP(&pingFlagOutput, "output", "o", "", "Output file to save results")
	pingScanCmd.Flags().IntVarP(&pingFlagTimeout, "timeout", "t", 1, "Ping timeout in seconds")
	pingScanCmd.Flags().IntVarP(&pingFlagDelay, "delay", "d", 100, "Delay between pings in milliseconds")
	pingScanCmd.Flags().IntVarP(&pingFlagCount, "count", "n", 3, "Number of ping attempts per IP")
	pingScanCmd.Flags().IntVarP(&pingFlagThreads, "threads", "T", 10, "Number of concurrent threads")
}

func pingHost(ip string, timeout, count int) bool {
	pinger, err := ping.NewPinger(ip)
	if err != nil {
		return false
	}
	pinger.Count = count
	pinger.Timeout = time.Duration(timeout) * time.Second
	err = pinger.Run()
	if err != nil {
		return false
	}
	stats := pinger.Statistics()
	return stats.PacketsRecv > 0
}

func pingScanRun(cmd *cobra.Command, args []string) {
	var ips []string

	if pingFlagCIDR != "" {
		ip, ipnet, err := net.ParseCIDR(pingFlagCIDR)
		if err != nil {
			fmt.Println("Invalid CIDR:", err)
			return
		}
		for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
			ips = append(ips, ip.String())
		}
	}

	if pingFlagFile != "" {
		file, err := os.Open(pingFlagFile)
		if err != nil {
			fmt.Println("Error opening file:", err)
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			ips = append(ips, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			fmt.Println("Error reading file:", err)
			return
		}
	}

	total := len(ips)
	results := make([]string, 0)
	successColor := color.New(color.FgGreen).SprintFunc()
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, pingFlagThreads)

	for i, ip := range ips {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, ip string) {
			defer wg.Done()
			defer func() { <-sem }()
			progress := float64(i+1) / float64(total) * 100
			fmt.Printf("\rScanning: %d/%d (%.2f%%)", i+1, total, progress)

			if pingHost(ip, pingFlagTimeout, pingFlagCount) {
				result := fmt.Sprintf("Ping successful: %s", ip)
				mu.Lock()
				fmt.Println()
				fmt.Println(successColor(result))
				results = append(results, result)
				mu.Unlock()
			}
			time.Sleep(time.Duration(pingFlagDelay) * time.Millisecond)
		}(i, ip)
	}
	wg.Wait()

	fmt.Println()

	if pingFlagOutput != "" {
		err := os.WriteFile(pingFlagOutput, []byte(strings.Join(results, "\n")), 0644)
		if err != nil {
			fmt.Println("Error writing to output file:", err)
		}
	}
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

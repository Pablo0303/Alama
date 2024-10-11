package cmd

import (
    "bufio"
    "fmt"
    "net"
    "os"
    "strings"
    "sync"
    "time"

    "github.com/go-ping/ping"
    "github.com/spf13/cobra"
    "github.com/fatih/color"
)

// directScanCmd represents the directScan command
var directScanCmd = &cobra.Command{
    Use:   "direct",
    Short: "Scan a range of IPs or a list of IPs/hosts directly",
    Run:   directScanRun,
}

var (
    directFlagCIDR    string
    directFlagFile    string
    directFlagOutput  string
    directFlagTimeout int
    directFlagDelay   int
    directFlagCount   int
    directFlagThreads int
)

func init() {
    rootCmd.AddCommand(directScanCmd)

    directScanCmd.Flags().StringVarP(&directFlagCIDR, "cidr", "c", "", "CIDR range to scan")
    directScanCmd.Flags().StringVarP(&directFlagFile, "file", "f", "", "File containing list of IPs/hosts to scan")
    directScanCmd.Flags().StringVarP(&directFlagOutput, "output", "o", "", "Output file to save results")
    directScanCmd.Flags().IntVarP(&directFlagTimeout, "timeout", "t", 1, "Scan timeout in seconds")
    directScanCmd.Flags().IntVarP(&directFlagDelay, "delay", "d", 250, "Delay between scans in milliseconds")
    directScanCmd.Flags().IntVarP(&directFlagCount, "count", "n", 1, "Number of scan attempts per IP")
    directScanCmd.Flags().IntVarP(&directFlagThreads, "threads", "T", 50, "Number of concurrent threads")
}

func directScanHost(ip string, timeout, count int) bool {
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

func directScanRun(cmd *cobra.Command, args []string) {
    var ips []string

    if directFlagCIDR != "" {
        ip, ipnet, err := net.ParseCIDR(directFlagCIDR)
        if err != nil {
            fmt.Println("Invalid CIDR:", err)
            return
        }
        for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incrementIP(ip) {
            ips = append(ips, ip.String())
        }
    }

    if directFlagFile != "" {
        file, err := os.Open(directFlagFile)
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
    found := 0
    var mu sync.Mutex
    var wg sync.WaitGroup
    sem := make(chan struct{}, directFlagThreads)
    green := color.New(color.FgGreen).SprintFunc()
    results := make([]string, 0)

    for i, ip := range ips {
        wg.Add(1)
        sem <- struct{}{}
        go func(i int, ip string) {
            defer wg.Done()
            defer func() { <-sem }()
            progress := float64(i+1) / float64(total) * 100

            if directScanHost(ip, directFlagTimeout, directFlagCount) {
                mu.Lock()
                found++
                results = append(results, ip)
                fmt.Printf("\n%s\n", green(ip)) // Mostrar IP en color verde en una línea independiente
                mu.Unlock()
            }

            // Actualizar la línea de progreso
            mu.Lock()
            logReplace(ip, found, total, i+1, progress)
            mu.Unlock()

            if directFlagDelay > 0 {
                time.Sleep(time.Duration(directFlagDelay) * time.Millisecond)
            }
        }(i, ip)
    }
    wg.Wait()

    // Asegurarse de que la línea final se muestre correctamente
    logReplace("", found, total, total, 100.00)

    if directFlagOutput != "" {
        err := os.WriteFile(directFlagOutput, []byte(strings.Join(results, "\n")), 0644)
        if err != nil {
            fmt.Println("Error writing to output file:", err)
        }
    }

    // Agregar un salto de línea al final para evitar el símbolo del sistema
    fmt.Print("\n")
}

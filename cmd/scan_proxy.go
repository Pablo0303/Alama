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

// proxyScanCmd represents the proxyScan command
var proxyScanCmd = &cobra.Command{
    Use:   "proxy",
    Short: "Scan a range of IPs or a list of IPs/hosts for proxies",
    Run:   proxyScanRun,
}

var (
    proxyFlagCIDR    string
    proxyFlagFile    string
    proxyFlagOutput  string
    proxyFlagTimeout int
    proxyFlagDelay   int
    proxyFlagCount   int
    proxyFlagThreads int
    proxyFlagProxy   string
)

func init() {
    rootCmd.AddCommand(proxyScanCmd)

    proxyScanCmd.Flags().StringVarP(&proxyFlagCIDR, "cidr", "c", "", "CIDR range to scan")
    proxyScanCmd.Flags().StringVarP(&proxyFlagFile, "file", "f", "", "File containing list of IPs/hosts to scan")
    proxyScanCmd.Flags().StringVarP(&proxyFlagOutput, "output", "o", "", "Output file to save results")
    proxyScanCmd.Flags().IntVarP(&proxyFlagTimeout, "timeout", "t", 1, "Scan timeout in seconds")
    proxyScanCmd.Flags().IntVarP(&proxyFlagDelay, "delay", "d", 250, "Delay between scans in milliseconds")
    proxyScanCmd.Flags().IntVarP(&proxyFlagCount, "count", "n", 1, "Number of scan attempts per IP")
    proxyScanCmd.Flags().IntVarP(&proxyFlagThreads, "threads", "T", 50, "Number of concurrent threads")
    proxyScanCmd.Flags().StringVarP(&proxyFlagProxy, "proxy", "x", "", "Proxy and port to use (e.g., 192.168.1.1:8080)")
}

func proxyScanHost(ip string, timeout, count int, proxy string) bool {
    // Implementar la lógica para usar el proxy si se especifica
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

func proxyScanRun(cmd *cobra.Command, args []string) {
    var ips []string

    if proxyFlagCIDR != "" {
        ip, ipnet, err := net.ParseCIDR(proxyFlagCIDR)
        if err != nil {
            fmt.Println("Invalid CIDR:", err)
            return
        }
        for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incrementIP(ip) {
            ips = append(ips, ip.String())
        }
    }

    if proxyFlagFile != "" {
        file, err := os.Open(proxyFlagFile)
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
    sem := make(chan struct{}, proxyFlagThreads)
    green := color.New(color.FgGreen).SprintFunc()
    results := make([]string, 0)

    for i, ip := range ips {
        wg.Add(1)
        sem <- struct{}{}
        go func(i int, ip string) {
            defer wg.Done()
            defer func() { <-sem }()
            progress := float64(i+1) / float64(total) * 100

            if proxyScanHost(ip, proxyFlagTimeout, proxyFlagCount, proxyFlagProxy) {
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

            if proxyFlagDelay > 0 {
                time.Sleep(time.Duration(proxyFlagDelay) * time.Millisecond)
            }
        }(i, ip)
    }
    wg.Wait()

    // Asegurarse de que la línea final se muestre correctamente
    logReplace("", found, total, total, 100.00)

    if proxyFlagOutput != "" {
        err := os.WriteFile(proxyFlagOutput, []byte(strings.Join(results, "\n")), 0644)
        if err != nil {
            fmt.Println("Error writing to output file:", err)
        }
    }

    // Agregar un salto de línea al final para evitar el símbolo del sistema
    fmt.Print("\n")
}

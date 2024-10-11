package cmd

import (
    "bufio"
    "fmt"
    "net"
    "os"
    "strings"
    "sync"
    "time"

    "github.com/spf13/cobra"
    "github.com/fatih/color"
)

// udpScanCmd represents the udpScan command
var udpScanCmd = &cobra.Command{
    Use:   "udp",
    Short: "Scan a range of IPs or a list of IPs/hosts for active UDP connections",
    Run:   udpScanRun,
}

var (
    udpFlagCIDR    string
    udpFlagFile    string
    udpFlagOutput  string
    udpFlagTimeout int
    udpFlagDelay   int
    udpFlagCount   int
    udpFlagThreads int
)

func init() {
    rootCmd.AddCommand(udpScanCmd)

    udpScanCmd.Flags().StringVarP(&udpFlagCIDR, "cidr", "c", "", "CIDR range to scan")
    udpScanCmd.Flags().StringVarP(&udpFlagFile, "file", "f", "", "File containing list of IPs/hosts to scan")
    udpScanCmd.Flags().StringVarP(&udpFlagOutput, "output", "o", "", "Output file to save results")
    udpScanCmd.Flags().IntVarP(&udpFlagTimeout, "timeout", "t", 1, "Scan timeout in seconds")
    udpScanCmd.Flags().IntVarP(&udpFlagDelay, "delay", "d", 250, "Delay between scans in milliseconds")
    udpScanCmd.Flags().IntVarP(&udpFlagCount, "count", "n", 1, "Number of scan attempts per IP")
    udpScanCmd.Flags().IntVarP(&udpFlagThreads, "threads", "T", 50, "Number of concurrent threads")
}

func udpScanHost(ip string, timeout, count int) bool {
    conn, err := net.Dial("udp", fmt.Sprintf("%s:53", ip))
    if err != nil {
        return false
    }
    defer conn.Close()

    message := []byte("UDP scan")
    for i := 0; i < count; i++ {
        _, err := conn.Write(message)
        if err != nil {
            return false
        }

        conn.SetReadDeadline(time.Now().Add(time.Duration(timeout) * time.Second))
        buffer := make([]byte, 1024)
        _, err = conn.Read(buffer)
        if err == nil {
            return true
        }
    }
    return false
}

func udpScanRun(cmd *cobra.Command, args []string) {
    var ips []string

    if udpFlagCIDR != "" {
        ip, ipnet, err := net.ParseCIDR(udpFlagCIDR)
        if err != nil {
            fmt.Println("Invalid CIDR:", err)
            return
        }
        for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incrementIP(ip) {
            ips = append(ips, ip.String())
        }
    }

    if udpFlagFile != "" {
        file, err := os.Open(udpFlagFile)
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
    sem := make(chan struct{}, udpFlagThreads)
    green := color.New(color.FgGreen).SprintFunc()
    results := make([]string, 0)

    for i, ip := range ips {
        wg.Add(1)
        sem <- struct{}{}
        go func(i int, ip string) {
            defer wg.Done()
            defer func() { <-sem }()
            progress := float64(i+1) / float64(total) * 100

            if udpScanHost(ip, udpFlagTimeout, udpFlagCount) {
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

            if udpFlagDelay > 0 {
                time.Sleep(time.Duration(udpFlagDelay) * time.Millisecond)
            }
        }(i, ip)
    }
    wg.Wait()

    // Asegurarse de que la línea final se muestre correctamente
    logReplace("", found, total, total, 100.00)

    if udpFlagOutput != "" {
        err := os.WriteFile(udpFlagOutput, []byte(strings.Join(results, "\n")), 0644)
        if err != nil {
            fmt.Println("Error writing to output file:", err)
        }
    }

    // Agregar un salto de línea al final para evitar el símbolo del sistema
    fmt.Print("\n")
}

package cmd

import (
    "bufio"
    "fmt"
    "net"
    "net/http"
    "os"
    "strings"
    "sync"
    "time"

    "github.com/go-ping/ping"
    "github.com/spf13/cobra"
    "github.com/fatih/color"
)

// cdnSslScanCmd represents the cdnSslScan command
var cdnSslScanCmd = &cobra.Command{
    Use:   "cdnssl",
    Short: "Scan a range of IPs or a list of IPs/hosts for CDN SSL",
    Run:   cdnSslScanRun,
}

var (
    cdnSslFlagCIDR    string
    cdnSslFlagFile    string
    cdnSslFlagOutput  string
    cdnSslFlagTimeout int
    cdnSslFlagDelay   int
    cdnSslFlagCount   int
    cdnSslFlagThreads int
)

func init() {
    rootCmd.AddCommand(cdnSslScanCmd)

    cdnSslScanCmd.Flags().StringVarP(&cdnSslFlagCIDR, "cidr", "c", "", "Rango CIDR para escanear")
    cdnSslScanCmd.Flags().StringVarP(&cdnSslFlagFile, "file", "f", "", "Archivo que contiene la lista de IPs/hosts para escanear")
    cdnSslScanCmd.Flags().StringVarP(&cdnSslFlagOutput, "output", "o", "", "Archivo de salida para guardar los resultados")
    cdnSslScanCmd.Flags().IntVarP(&cdnSslFlagTimeout, "timeout", "t", 1, "Tiempo de espera del escaneo en segundos")
    cdnSslScanCmd.Flags().IntVarP(&cdnSslFlagDelay, "delay", "d", 250, "Retraso entre escaneos en milisegundos")
    cdnSslScanCmd.Flags().IntVarP(&cdnSslFlagCount, "count", "n", 1, "Número de intentos de escaneo por IP")
    cdnSslScanCmd.Flags().IntVarP(&cdnSslFlagThreads, "threads", "T", 50, "Número de hilos concurrentes")
}

func cdnSslScanHost(ip string, timeout, count int) (bool, string, string) {
    pinger, err := ping.NewPinger(ip)
    if err != nil {
        return false, "", ""
    }
    pinger.Count = count
    pinger.Timeout = time.Duration(timeout) * time.Second
    err = pinger.Run()
    if err != nil {
        return false, "", ""
    }
    stats := pinger.Statistics()
    if stats.PacketsRecv > 0 {
        // Realizar una solicitud HTTP para obtener la información del servidor y el código de estado
        url := fmt.Sprintf("http://%s", ip)
        client := &http.Client{
            Timeout: time.Duration(timeout) * time.Second,
        }
        resp, err := client.Get(url)
        if err != nil {
            return true, "", ""
        }
        defer resp.Body.Close()
        server := resp.Header.Get("Server")
        status := resp.Status
        return true, server, status
    }
    return false, "", ""
}

func cdnSslScanRun(cmd *cobra.Command, args []string) {
    var ips []string

    if cdnSslFlagCIDR != "" {
        ip, ipnet, err := net.ParseCIDR(cdnSslFlagCIDR)
        if err != nil {
            fmt.Println("Rango CIDR inválido:", err)
            return
        }
        for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incrementIP(ip) {
            ips = append(ips, ip.String())
        }
    }

    if cdnSslFlagFile != "" {
        file, err := os.Open(cdnSslFlagFile)
        if err != nil {
            fmt.Println("Error al abrir el archivo:", err)
            return
        }
        defer file.Close()

        scanner := bufio.NewScanner(file)
        for scanner.Scan() {
            ips = append(ips, scanner.Text())
        }
        if err := scanner.Err(); err != nil {
            fmt.Println("Error al leer el archivo:", err)
            return
        }
    }

    total := len(ips)
    found := 0
    var mu sync.Mutex
    var wg sync.WaitGroup
    sem := make(chan struct{}, cdnSslFlagThreads)
    green := color.New(color.FgGreen).SprintFunc()
    results := make([]string, 0)

    for i, ip := range ips {
        wg.Add(1)
        sem <- struct{}{}
        go func(i int, ip string) {
            defer wg.Done()
            defer func() { <-sem }()
            progress := float64(i+1) / float64(total) * 100

            success, server, status := cdnSslScanHost(ip, cdnSslFlagTimeout, cdnSslFlagCount)
            if success {
                mu.Lock()
                found++
                result := fmt.Sprintf("%s - %s - %s", ip, server, status)
                results = append(results, result)
                fmt.Printf("\n%s\n", green(result)) // Mostrar IP, servidor y estado en color verde en una línea independiente
                mu.Unlock()
            }

            // Actualizar la línea de progreso
            mu.Lock()
            logReplace(ip, found, total, i+1, progress)
            mu.Unlock()

            if cdnSslFlagDelay > 0 {
                time.Sleep(time.Duration(cdnSslFlagDelay) * time.Millisecond)
            }
        }(i, ip)
    }
    wg.Wait()

    // Asegurarse de que la línea final se muestre correctamente
    logReplace("", found, total, total, 100.00)

    if cdnSslFlagOutput != "" {
        err := os.WriteFile(cdnSslFlagOutput, []byte(strings.Join(results, "\n")), 0644)
        if err != nil {
            fmt.Println("Error al escribir en el archivo de salida:", err)
        }
    }

    // Agregar un salto de línea al final para evitar el símbolo del sistema
    fmt.Print("\n")
}

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

// sniScanCmd represents the sniScan command
var sniScanCmd = &cobra.Command{
    Use:   "sni",
    Short: "Scan a range of IPs or a list of IPs/hosts for SNI",
    Run:   sniScanRun,
}

var (
    sniFlagCIDR    string
    sniFlagFile    string
    sniFlagOutput  string
    sniFlagTimeout int
    sniFlagDelay   int
    sniFlagCount   int
    sniFlagThreads int
)

func init() {
    rootCmd.AddCommand(sniScanCmd)

    sniScanCmd.Flags().StringVarP(&sniFlagCIDR, "cidr", "c", "", "Rango CIDR para escanear")
    sniScanCmd.Flags().StringVarP(&sniFlagFile, "file", "f", "", "Archivo que contiene la lista de IPs/hosts para escanear")
    sniScanCmd.Flags().StringVarP(&sniFlagOutput, "output", "o", "", "Archivo de salida para guardar los resultados")
    sniScanCmd.Flags().IntVarP(&sniFlagTimeout, "timeout", "t", 1, "Tiempo de espera del escaneo en segundos")
    sniScanCmd.Flags().IntVarP(&sniFlagDelay, "delay", "d", 250, "Retraso entre escaneos en milisegundos")
    sniScanCmd.Flags().IntVarP(&sniFlagCount, "count", "n", 1, "Número de intentos de escaneo por IP")
    sniScanCmd.Flags().IntVarP(&sniFlagThreads, "threads", "T", 50, "Número de hilos concurrentes")
}

func sniScanHost(ip string, timeout, count int) (bool, string, string) {
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

func sniScanRun(cmd *cobra.Command, args []string) {
    var ips []string

    if sniFlagCIDR != "" {
        ip, ipnet, err := net.ParseCIDR(sniFlagCIDR)
        if err != nil {
            fmt.Println("Rango CIDR inválido:", err)
            return
        }
        for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incrementIP(ip) {
            ips = append(ips, ip.String())
        }
    }

    if sniFlagFile != "" {
        file, err := os.Open(sniFlagFile)
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
    sem := make(chan struct{}, sniFlagThreads)
    green := color.New(color.FgGreen).SprintFunc()
    results := make([]string, 0)

    for i, ip := range ips {
        wg.Add(1)
        sem <- struct{}{}
        go func(i int, ip string) {
            defer wg.Done()
            defer func() { <-sem }()
            progress := float64(i+1) / float64(total) * 100

            success, server, status := sniScanHost(ip, sniFlagTimeout, sniFlagCount)
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

            if sniFlagDelay > 0 {
                time.Sleep(time.Duration(sniFlagDelay) * time.Millisecond)
            }
        }(i, ip)
    }
    wg.Wait()

    // Asegurarse de que la línea final se muestre correctamente
    logReplace("", found, total, total, 100.00)

    if sniFlagOutput != "" {
        err := os.WriteFile(sniFlagOutput, []byte(strings.Join(results, "\n")), 0644)
        if err != nil {
            fmt.Println("Error al escribir en el archivo de salida:", err)
        }
    }

    // Agregar un salto de línea al final para evitar el símbolo del sistema
    fmt.Print("\n")
}

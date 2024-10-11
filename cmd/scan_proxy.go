package cmd

import (
    "bufio"
    "fmt"
    "net"
    "net/http"
    "net/url"
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

    proxyScanCmd.Flags().StringVarP(&proxyFlagCIDR, "cidr", "c", "", "Rango CIDR para escanear")
    proxyScanCmd.Flags().StringVarP(&proxyFlagFile, "file", "f", "", "Archivo que contiene la lista de IPs/hosts para escanear")
    proxyScanCmd.Flags().StringVarP(&proxyFlagOutput, "output", "o", "", "Archivo de salida para guardar los resultados")
    proxyScanCmd.Flags().IntVarP(&proxyFlagTimeout, "timeout", "t", 1, "Tiempo de espera del escaneo en segundos")
    proxyScanCmd.Flags().IntVarP(&proxyFlagDelay, "delay", "d", 250, "Retraso entre escaneos en milisegundos")
    proxyScanCmd.Flags().IntVarP(&proxyFlagCount, "count", "n", 1, "Número de intentos de escaneo por IP")
    proxyScanCmd.Flags().IntVarP(&proxyFlagThreads, "threads", "T", 50, "Número de hilos concurrentes")
    proxyScanCmd.Flags().StringVarP(&proxyFlagProxy, "proxy", "x", "", "Proxy y puerto a usar (ej., 192.168.1.1:8080)")
}

func proxyScanHost(ip string, timeout, count int, proxy string) (bool, string, string) {
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
        urlStr := fmt.Sprintf("http://%s", ip)
        client := &http.Client{
            Timeout: time.Duration(timeout) * time.Second,
        }
        if proxy != "" {
            proxyURL, err := url.Parse(fmt.Sprintf("http://%s", proxy))
            if err != nil {
                return true, "", ""
            }
            client.Transport = &http.Transport{
                Proxy: http.ProxyURL(proxyURL),
            }
        }
        resp, err := client.Get(urlStr)
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

func proxyScanRun(cmd *cobra.Command, args []string) {
    var ips []string

    if proxyFlagCIDR != "" {
        ip, ipnet, err := net.ParseCIDR(proxyFlagCIDR)
        if err != nil {
            fmt.Println("Rango CIDR inválido:", err)
            return
        }
        for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incrementIP(ip) {
            ips = append(ips, ip.String())
        }
    }

    if proxyFlagFile != "" {
        file, err := os.Open(proxyFlagFile)
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

            success, server, status := proxyScanHost(ip, proxyFlagTimeout, proxyFlagCount, proxyFlagProxy)
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
            fmt.Println("Error al escribir en el archivo de salida:", err)
        }
    }

    // Agregar un salto de línea al final para evitar el símbolo del sistema
    fmt.Print("\n")
}

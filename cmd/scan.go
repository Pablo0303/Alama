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

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
    Use:   "scan",
    Short: "Escaneo general de IPs/hosts usando ping",
    Long: `Escanea un rango de IPs o una lista de IPs/hosts para determinar si están activos.
    
Opciones:
  -c, --cidr string       Rango CIDR para escanear
  -f, --file string       Archivo que contiene la lista de IPs/hosts para escanear
  -o, --output string     Archivo de salida para guardar los resultados
  -t, --timeout int       Tiempo de espera del escaneo en segundos (por defecto 1)
  -d, --delay int         Retraso entre escaneos en milisegundos (por defecto 250)
  -n, --count int         Número de intentos de escaneo por IP (por defecto 1)
  -T, --threads int       Número de hilos concurrentes (por defecto 50)
`,
    Run: scanRun,
}

var (
    scanFlagCIDR    string
    scanFlagFile    string
    scanFlagOutput  string
    scanFlagTimeout int
    scanFlagDelay   int
    scanFlagCount   int
    scanFlagThreads int
)

func init() {
    rootCmd.AddCommand(scanCmd)

    scanCmd.Flags().StringVarP(&scanFlagCIDR, "cidr", "c", "", "Rango CIDR para escanear")
    scanCmd.Flags().StringVarP(&scanFlagFile, "file", "f", "", "Archivo que contiene la lista de IPs/hosts para escanear")
    scanCmd.Flags().StringVarP(&scanFlagOutput, "output", "o", "", "Archivo de salida para guardar los resultados")
    scanCmd.Flags().IntVarP(&scanFlagTimeout, "timeout", "t", 1, "Tiempo de espera del escaneo en segundos")
    scanCmd.Flags().IntVarP(&scanFlagDelay, "delay", "d", 250, "Retraso entre escaneos en milisegundos")
    scanCmd.Flags().IntVarP(&scanFlagCount, "count", "n", 1, "Número de intentos de escaneo por IP")
    scanCmd.Flags().IntVarP(&scanFlagThreads, "threads", "T", 50, "Número de hilos concurrentes")
}

func scanHost(ip string, timeout, count int) bool {
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

func scanRun(cmd *cobra.Command, args []string) {
    var ips []string

    if scanFlagCIDR != "" {
        ip, ipnet, err := net.ParseCIDR(scanFlagCIDR)
        if err != nil {
            fmt.Println("Rango CIDR inválido:", err)
            return
        }
        for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incrementIP(ip) {
            ips = append(ips, ip.String())
        }
    }

    if scanFlagFile != "" {
        file, err := os.Open(scanFlagFile)
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
    sem := make(chan struct{}, scanFlagThreads)
    green := color.New(color.FgGreen).SprintFunc()
    results := make([]string, 0)

    for i, ip := range ips {
        wg.Add(1)
        sem <- struct{}{}
        go func(i int, ip string) {
            defer wg.Done()
            defer func() { <-sem }()
            progress := float64(i+1) / float64(total) * 100

            if scanHost(ip, scanFlagTimeout, scanFlagCount) {
                mu.Lock()
                found++
                results = append(results, ip)
                fmt.Println(green(ip)) // Mostrar IP en color verde
                mu.Unlock()
            }

            // Actualizar la línea de progreso
            mu.Lock()
            logReplace(ip, found, total, i+1, progress)
            mu.Unlock()

            if scanFlagDelay > 0 {
                time.Sleep(time.Duration(scanFlagDelay) * time.Millisecond)
            }
        }(i, ip)
    }
    wg.Wait()

    // Asegurarse de que la línea final se muestre correctamente
    logReplace("", found, total, total, 100.00)

    if scanFlagOutput != "" {
        err := os.WriteFile(scanFlagOutput, []byte(strings.Join(results, "\n")), 0644)
        if err != nil {
            fmt.Println("Error al escribir en el archivo de salida:", err)
        }
    }

    // Agregar un salto de línea al final para evitar el símbolo del sistema
    fmt.Print("\n")
}

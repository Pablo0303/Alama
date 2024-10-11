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

	pingScanCmd.Flags().StringVarP(&pingFlagCIDR, "cidr", "c", "", "Rango CIDR para escanear")
	pingScanCmd.Flags().StringVarP(&pingFlagFile, "file", "f", "", "Archivo que contiene la lista de IPs/hosts para escanear")
	pingScanCmd.Flags().StringVarP(&pingFlagOutput, "output", "o", "", "Archivo de salida para guardar los resultados")
	pingScanCmd.Flags().IntVarP(&pingFlagTimeout, "timeout", "t", 1, "Tiempo de espera del escaneo en segundos")
	pingScanCmd.Flags().IntVarP(&pingFlagDelay, "delay", "d", 250, "Retraso entre escaneos en milisegundos")
	pingScanCmd.Flags().IntVarP(&pingFlagCount, "count", "n", 1, "Número de intentos de escaneo por IP")
	pingScanCmd.Flags().IntVarP(&pingFlagThreads, "threads", "T", 50, "Número de hilos concurrentes")
}

func pingScanHost(ip string, timeout, count int) bool {
	pinger, err := ping.NewPinger(ip)
	if err != nil {
		fmt.Println("Error al crear el pinger:", err)
		return false
	}
	pinger.Count = count
	pinger.Timeout = time.Duration(timeout) * time.Second
	err = pinger.Run()
	if err != nil {
		fmt.Println("Error al ejecutar el pinger:", err)
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
			fmt.Println("Rango CIDR inválido:", err)
			return
		}
		for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incrementIP(ip) {
			ips = append(ips, ip.String())
		}
	}

	if pingFlagFile != "" {
		file, err := os.Open(pingFlagFile)
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
	sem := make(chan struct{}, pingFlagThreads)
	green := color.New(color.FgGreen).SprintFunc()
	results := make([]string, 0)

	for i, ip := range ips {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, ip string) {
			defer wg.Done()
			defer func() { <-sem }()
			progress := float64(i+1) / float64(total) * 100

			if pingScanHost(ip, pingFlagTimeout, pingFlagCount) {
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

			if pingFlagDelay > 0 {
				time.Sleep(time.Duration(pingFlagDelay) * time.Millisecond)
			}
		}(i, ip)
	}
	wg.Wait()

	// Asegurarse de que la línea final se muestre correctamente
	logReplace("", found, total, total, 100.00)

	if pingFlagOutput != "" {
		err := os.WriteFile(pingFlagOutput, []byte(strings.Join(results, "\n")), 0644)
		if err != nil {
			fmt.Println("Error al escribir en el archivo de salida:", err)
		}
	}

	// Agregar un salto de línea al final para evitar el símbolo del sistema
	fmt.Print("\n")
}

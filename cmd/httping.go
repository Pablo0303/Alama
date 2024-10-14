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

	"github.com/spf13/cobra"
	"github.com/fatih/color"
)

// httpingCmd representa el comando httping
var httpingCmd = &cobra.Command{
	Use:   "httping",
	Short: "Escaneo HTTP/HTTPS de IPs/hosts",
	Long:  `Escanea un rango de IPs o una lista de IPs/hosts para determinar su estado HTTP.`,
	Run:   httpingRun,
}

var (
	httpingFlagCIDR     string
	httpingFlagFile     string
	httpingFlagOutput   string
	httpingFlagTimeout  int
	httpingFlagDelay    int
	httpingFlagCount    int
	httpingFlagThreads  int
	httpingFlagStatus   string
	httpingFlagProxy    string
	httpingFlagHTTPVerb string
)

func init() {
	rootCmd.AddCommand(httpingCmd)

	httpingCmd.Flags().StringVarP(&httpingFlagCIDR, "cidr", "c", "", "Rango CIDR para escanear")
	httpingCmd.Flags().StringVarP(&httpingFlagFile, "file", "f", "", "Archivo que contiene la lista de IPs/hosts para escanear")
	httpingCmd.Flags().StringVarP(&httpingFlagOutput, "output", "o", "", "Archivo de salida para guardar los resultados")
	httpingCmd.Flags().IntVarP(&httpingFlagTimeout, "timeout", "t", 1, "Tiempo de espera del escaneo en segundos")
	httpingCmd.Flags().IntVarP(&httpingFlagDelay, "delay", "d", 250, "Retraso entre escaneos en milisegundos")
	httpingCmd.Flags().IntVarP(&httpingFlagCount, "count", "n", 1, "Número de intentos de escaneo por IP")
	httpingCmd.Flags().IntVarP(&httpingFlagThreads, "threads", "T", 50, "Número de hilos concurrentes")
	httpingCmd.Flags().StringVarP(&httpingFlagStatus, "status", "s", "", "Códigos de estado HTTP a mostrar (ej. 200,500)")
	httpingCmd.Flags().StringVarP(&httpingFlagProxy, "proxy", "x", "", "Proxy y puerto a usar (ej., 192.168.1.1:8080)")
	httpingCmd.Flags().StringVarP(&httpingFlagHTTPVerb, "httpverb", "v", "GET", "HTTP Verb: Only GET or HEAD supported at the moment")
}

func httpingRun(cmd *cobra.Command, args []string) {
	var ips []string

	// Procesar el rango CIDR
	if httpingFlagCIDR != "" {
		ip, ipnet, err := net.ParseCIDR(httpingFlagCIDR)
		if err != nil {
			fmt.Println("Rango CIDR inválido:", err)
			return
		}
		for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incrementIP(ip) {
			ips = append(ips, ip.String())
		}
	}

	// Procesar el archivo de hosts
	if httpingFlagFile != "" {
		file, err := os.Open(httpingFlagFile)
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

	// Escaneo de hosts
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, httpingFlagThreads)
	green := color.New(color.FgGreen).SprintFunc() // Se utiliza para dar formato verde
	results := make([]string, 0)
	validResultsCount := 0 // Contador de resultados válidos

	// Mostrar el progreso en tiempo real
	totalIPs := len(ips)

	for index, ip := range ips {
		wg.Add(1)
		sem <- struct{}{}
		go func(ip string, index int) {
			defer wg.Done()
			defer func() { <-sem }()

			// Hacer la solicitud HTTP
			statusCode := scanHTTP(ip, httpingFlagTimeout)

			mu.Lock() // Asegurarse de que no haya interferencia al acceder a `results`
			if statusCode != 0 {
				if httpingFlagStatus == "" || strings.Contains(httpingFlagStatus, fmt.Sprint(statusCode)) {
					// Solo agregar si el estado coincide
					results = append(results, fmt.Sprintf("%-20s %s", ip, green(fmt.Sprint(statusCode)))) // Mostrar IP y estado en verde
					validResultsCount++ // Incrementar el contador de resultados válidos
				}
			}

			// Mostrar el progreso en función del índice de la IP escaneada
			fmt.Printf("\rEscaneando %d/%d (F:%d) %.2f%%", index+1, totalIPs, validResultsCount, float64(index+1)/float64(totalIPs)*100)
			mu.Unlock()

			if httpingFlagDelay > 0 {
				time.Sleep(time.Duration(httpingFlagDelay) * time.Millisecond)
			}
		}(ip, index)
	}
	wg.Wait()

	// Asegurarse de que el progreso se complete al final
	mu.Lock()
	fmt.Printf("\rEscaneando %d/%d (F:%d) 100.00%%\n", totalIPs, totalIPs, validResultsCount)
	mu.Unlock()

	// Imprimir resultados en la consola
	if len(results) > 0 {
		for _, result := range results {
			fmt.Println(result) // Mostrar solo los resultados que coinciden con -s
		}
	} else {
		// Si no hay resultados, imprimir un mensaje
		fmt.Println("\nNo se encontraron resultados que coincidan con los criterios dados.")
	}

	// Guardar resultados en el archivo de salida si se solicita
	if httpingFlagOutput != "" {
		err := os.WriteFile(httpingFlagOutput, []byte(strings.Join(results, "\n")), 0644)
		if err != nil {
			fmt.Println("Error al escribir en el archivo de salida:", err)
		}
	}
}

// scanHTTP realiza una solicitud HTTP y devuelve el código de estado.
func scanHTTP(ip string, timeout int) int {
	client := http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	// Configurar proxy si se especifica
	if httpingFlagProxy != "" {
		proxyURL, err := url.Parse(httpingFlagProxy) // Usar el paquete url para parsear
		if err == nil {
			transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
			client.Transport = transport
		}
	}

	var req *http.Request
	var err error

	// Crear la solicitud según el verbo HTTP especificado
	if httpingFlagHTTPVerb == "HEAD" {
		req, err = http.NewRequest("HEAD", "http://"+ip, nil)
	} else {
		req, err = http.NewRequest("GET", "http://"+ip, nil)
	}

	if err != nil {
		return 0 // Retornar 0 si hay error en la creación de la solicitud
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0 // Retornar 0 si hay error en la solicitud
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

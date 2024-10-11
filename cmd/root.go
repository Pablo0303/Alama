package cmd

import (
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
    "os"
)

var rootCmd = &cobra.Command{
    Use:   "Alama",
    Short: "Alama es una herramienta de escaneo de redes",
    Long: `Alama es una herramienta versátil de escaneo de redes que soporta varios tipos de escaneos.
    
Módulos disponibles:
  - scan: Escaneo general de IPs/hosts usando ping.
  - cdnssl: Escaneo de SSL de CDN.
  - direct: Escaneo directo de IPs/hosts.
  - proxy: Escaneo de proxies activos.
  - sni: Escaneo de Server Name Indication (SNI).
  - udp: Escaneo de conexiones UDP activas.

Opciones:
  -c, --cidr string       Rango CIDR para escanear
  -f, --file string       Archivo que contiene la lista de IPs/hosts para escanear
  -o, --output string     Archivo de salida para guardar los resultados
  -t, --timeout int       Tiempo de espera del escaneo en segundos (por defecto 1)
  -d, --delay int         Retraso entre escaneos en milisegundos (por defecto 250)
  -n, --count int         Número de intentos de escaneo por IP (por defecto 1)
  -T, --threads int       Número de hilos concurrentes (por defecto 50)
  -x, --proxy string      Proxy y puerto a usar (ej., 192.168.1.1:8080)

Ejemplos de uso:
  - Escaneo general de IPs/hosts:
    ./Alama scan -c 192.168.1.0/24
    ./Alama scan -f ips.txt
    ./Alama scan --target vps.example.com

  - Escaneo de SSL de CDN:
    ./Alama cdnssl -c 192.168.1.0/24
    ./Alama cdnssl -f ips.txt
    ./Alama cdnssl --target vps.example.com
    ./Alama cdnssl --proxy-filename cf.txt --target ws.example.com

  - Escaneo directo de IPs/hosts:
    ./Alama direct -c 192.168.1.0/24
    ./Alama direct -f ips.txt
    ./Alama direct --target vps.example.com

  - Escaneo de proxies:
    ./Alama proxy -c 192.168.1.0/24
    ./Alama proxy -f ips.txt
    ./Alama proxy --target vps.example.com
    ./Alama proxy --http -f personal.txt -x 192.168.1.1:8080

  - Escaneo de SNI:
    ./Alama sni -c 192.168.1.0/24
    ./Alama sni -f ips.txt
    ./Alama sni --target vps.example.com

  - Escaneo de UDP:
    ./Alama udp -c 192.168.1.0/24
    ./Alama udp -f ips.txt
    ./Alama udp --target vps.example.com
`,
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}

func init() {
    cobra.OnInitialize(initConfig)
    rootCmd.PersistentFlags().StringP("config", "c", "", "archivo de configuración (por defecto es $HOME/.alama.yaml)")
    viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))

    rootCmd.AddCommand(scanCmd)
    rootCmd.AddCommand(cdnSslScanCmd)
    rootCmd.AddCommand(directScanCmd)
    rootCmd.AddCommand(proxyScanCmd)
    rootCmd.AddCommand(sniScanCmd)
    rootCmd.AddCommand(udpScanCmd)
}

func initConfig() {
    viper.AutomaticEnv()
}

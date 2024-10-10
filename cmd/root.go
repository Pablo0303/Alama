package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// Definir las variables para las opciones globales
var (
	ports        []string
	timeout      int
	threads      int
	outputFile   string
	proxy        string
	server       string
	payload      string
	httpScan     bool
	httpsScan    bool
	httpMethods  []string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "Alama",
	Short: "Esta herramienta esta libre de uso para aprendizaje sobre redes.",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.Alama.yaml)")

	// Agregar las nuevas opciones globales
	rootCmd.PersistentFlags().StringSliceVarP(&ports, "port", "p", nil, "Ports to scan")
	rootCmd.PersistentFlags().IntVarP(&timeout, "timeout", "t", 10, "Timeout for scans (in seconds)")
	rootCmd.PersistentFlags().IntVarP(&threads, "threads", "T", 10, "Number of threads")
	rootCmd.PersistentFlags().StringVarP(&outputFile, "outputfile", "o", "", "Output file")
	rootCmd.PersistentFlags().StringVarP(&proxy, "proxy", "x", "", "Proxy to use (e.g., squid proxy 172.22.2.38)")
	rootCmd.PersistentFlags().StringVarP(&server, "server", "s", "", "Websocket server domain/ip")
	rootCmd.PersistentFlags().StringVarP(&payload, "payload", "d", "", "Custom payload for scans")
	rootCmd.PersistentFlags().BoolVar(&httpScan, "http", false, "Perform HTTP scan")
	rootCmd.PersistentFlags().BoolVar(&httpsScan, "https", false, "Perform HTTPS scan")
	rootCmd.PersistentFlags().StringSliceVar(&httpMethods, "re", nil, "HTTP/HTTPS request methods to use for the sweep scan")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

var (
	colorD1 = color.New()
	colorB1 = color.New(color.FgHiBlack)
	colorW1 = color.New(color.FgWhite, color.Bold)
	colorG1 = color.New(color.FgGreen, color.Bold)
	colorC1 = color.New(color.FgCyan, color.Bold)
	colorY1 = color.New(color.FgYellow, color.Bold)
)

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".Alama" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".Alama")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}


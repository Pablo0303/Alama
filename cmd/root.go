package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd representa el comando base cuando se llama sin subcomandos
var rootCmd = &cobra.Command{
	Use:   "Alama",
	Short: "Esta herramienta está hecha para el conocimiento",
}

// Execute añade todos los subcomandos al comando raíz y configura las banderas apropiadamente.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	// Aquí defines tus banderas y configuraciones.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Archivo de configuración (predeterminado es $HOME/.Alama.yaml)")
	rootCmd.Flags().BoolP("toggle", "t", false, "Mensaje de ayuda para toggle")
}

var (
	colorD1 = color.New()
	colorB1 = color.New(color.FgHiBlack)
	colorW1 = color.New(color.FgWhite, color.Bold)
	colorG1 = color.New(color.FgGreen, color.Bold)
	colorC1 = color.New(color.FgCyan, color.Bold)
	colorY1 = color.New(color.FgYellow, color.Bold)
)

// initConfig lee el archivo de configuración y las variables de entorno si están configuradas.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".Alama")
	}

	viper.AutomaticEnv() // leer variables de entorno que coinciden

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Usando archivo de configuración:", viper.ConfigFileUsed())
	}
}

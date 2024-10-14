package cmd

import (
    "fmt"
    "net"
    "github.com/fatih/color"
    terminal "github.com/wayneashleyberry/terminal-dimensions"
)

// logReplace imprime el estado del escaneo en una sola línea.
func logReplace(ip string, found, total, current int, progress float64) {
    green := color.New(color.FgGreen).SprintFunc()
    s := fmt.Sprintf("Escaneando: %s F:%s %d/%d %.2f%%", ip, green(found), current, total, progress)

    termWidth, _, err := terminal.Dimensions()
    if err == nil {
        w := int(termWidth) - 3
        if len(s) >= w {
            s = s[:w] + "..."
        }
    }

    fmt.Print("\r\033[2K", s, "\r")
}

// incrementIP toma una dirección IP y la incrementa en 1.
func incrementIP(ip net.IP) net.IP {
    for j := len(ip) - 1; j >= 0; j-- {
        ip[j]++
        if ip[j] > 0 {
            break
        }
    }
    return ip // Retorna la IP incrementada
}
